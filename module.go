package wasmedge

// #include "shims.h"
import "C"

import (
	"fmt"
	"runtime"
	"runtime/cgo"
	"sync"
	"unsafe"
)

// moduleDataTokens is intentionally separate from externRefTokens. A token
// valid in one domain must remain foreign in the other, even though both are
// represented by C pointer addresses.
var moduleDataTokens sync.Map // map[unsafe.Pointer]*moduleDataState

type moduleDataState struct {
	mu     sync.RWMutex
	handle cgo.Handle
	token  unsafe.Pointer
	closed bool
}

func (s *moduleDataState) value() (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return nil, false
	}
	return s.handle.Value(), true
}

func (s *moduleDataState) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	s.handle.Delete()
	deleteOpaqueToken(s.token)
	s.handle = 0
	s.token = nil
}

// finalizeModuleData releases a registered token exactly once. It is called
// by the engine's native module destructor and by the constructor failure
// path, where no native destructor exists.
func finalizeModuleData(token unsafe.Pointer) {
	raw, ok := moduleDataTokens.LoadAndDelete(token)
	if !ok {
		return
	}
	state, ok := raw.(*moduleDataState)
	if !ok {
		return
	}
	state.release()
}

// Module is a module instance: either a host module built with NewModule
// (a bag of host functions/tables/memories/globals to import from), or an
// instantiated WASM module obtained from an Executor, VM, Store or plugin.
type Module struct {
	ptr  *C.WasmEdge_ModuleInstanceContext
	life lifetime
	// adopted keeps the Go closures of host functions transferred into this
	// module alive until the module itself is closed. Registration is not
	// synchronized: build a host module from one goroutine, like every
	// other constructor-phase API here.
	adopted []unsafe.Pointer
	// roots conservatively retain Go-backed references passed into guest code.
	// referenceStates are shared by every table/global view of this module.
	roots           persistentRoots
	referenceMu     sync.Mutex
	referenceStates map[referenceObject]*referenceRoots
	dependencies    []func()
	// dependencyModules mirrors the identities behind dependencies so result
	// rooting can reject a Go lease edge that would point back through an
	// imported host module.
	dependencyModules []*Module
	registrations     []*storeState
	// wasi is non-nil only when this wrapper has trusted provenance from a
	// WASI constructor/accessor. Borrowed aliases share the same state.
	wasi *wasiModuleState
}

// NewModule creates an empty named host module. Fill it with AddFunction
// (and friends), then register it into a VM, Executor or Store. This
// allocation-only constructor is semantically infallible and panics only if
// WasmEdge cannot allocate the native module.
func NewModule(name string) *Module {
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.WasmEdge_ModuleInstanceCreate(cname)
	if ptr == nil {
		panic("wasmedge: native module allocation failed")
	}
	return ownedModule(ptr)
}

// NewModuleWithData creates a named host module carrying arbitrary Go host
// data, retrievable via HostData from any borrowed view of the module. The
// pin on data is released when the module instance is destroyed, wherever
// that happens (Close here, or inside the engine). Native allocation failure
// is treated as an unrecoverable panic, like NewModule.
func NewModuleWithData(name string, data any) *Module {
	if data == nil {
		return NewModule(name)
	}
	h := cgo.NewHandle(data)
	token := newOpaqueToken()
	if token == nil {
		h.Delete()
		panic("wasmedge: native module-data token allocation failed")
	}
	state := &moduleDataState{handle: h, token: token}
	moduleDataTokens.Store(token, state)
	module := newModuleWithDataToken(name, token)
	if module == nil {
		finalizeModuleData(token)
		panic("wasmedge: native module allocation failed")
	}
	return module
}

// newModuleWithDataToken is the raw C constructor shared with provenance
// tests. Ownership of token remains with the registry that recognizes it;
// the native finalizer ignores foreign tokens.
func newModuleWithDataToken(name string, token unsafe.Pointer) *Module {
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.wasmedgego_moduleCreateWithData(cname, token)
	if ptr == nil {
		return nil
	}
	return ownedModule(ptr)
}

func ownedModule(ptr *C.WasmEdge_ModuleInstanceContext) *Module {
	m := &Module{ptr: ptr}
	// Native teardown and adopted host-function handle release are both
	// explicit-only. A leaked wrapper is reported under wasmedge_debug but
	// deliberately leaks rather than risking nondeterministic use-after-free.
	arm(m, &m.life, "Module", func() { C.WasmEdge_ModuleInstanceDelete(ptr) })
	return m
}

func borrowedModule(ptr *C.WasmEdge_ModuleInstanceContext, owner any) *Module {
	if ptr == nil {
		return nil
	}
	return &Module{
		ptr:  ptr,
		life: borrowed(owner),
		wasi: inheritedWASIState(ptr, owner),
	}
}

func (m *Module) assertAlive() { m.life.assertAlive("Module") }

func (m *Module) acquireLease() (func(), error) {
	return m.life.acquire("Module")
}

func (m *Module) retainReference(root any) bool {
	if root == nil {
		return true
	}
	if m.life.isOwned() {
		if ownerChainContainsModule(root, uintptr(unsafe.Pointer(m.ptr))) {
			return true
		}
		return m.roots.retain(root) == nil
	}
	if owner, ok := m.life.owner.(referenceRetainer); ok {
		return owner.retainReference(root)
	}
	return false
}

func (m *Module) prepareInvocationReferences(
	values []Value,
) (func(bool), error) {
	if !m.life.isOwned() {
		if owner, ok := m.life.owner.(invocationReferenceRetainer); ok {
			return owner.prepareInvocationReferences(values)
		}
		return nil, fmt.Errorf("retain guest reference parameter: %w", ErrOwnership)
	}

	owners := valueOwners(values)
	filtered := make([]any, 0, len(owners))
	module := uintptr(unsafe.Pointer(m.ptr))
	for _, owner := range owners {
		if ownerChainContainsModule(owner, module) {
			continue
		}
		filtered = append(filtered, owner)
	}
	finish, err := m.roots.prepare(filtered)
	if err != nil {
		return nil, fmt.Errorf("retain guest reference parameter: %w", err)
	}
	return finish, nil
}

func (m *Module) referenceState(object referenceObject) (*referenceRoots, bool) {
	if !m.life.isOwned() {
		return referenceState(m.life.owner, object)
	}
	m.referenceMu.Lock()
	defer m.referenceMu.Unlock()
	if m.referenceStates == nil {
		m.referenceStates = make(map[referenceObject]*referenceRoots)
	}
	state := m.referenceStates[object]
	if state == nil {
		state = &referenceRoots{container: m}
		m.referenceStates[object] = state
	}
	return state, true
}

func (m *Module) adoptReferenceState(object referenceObject, state *referenceRoots) error {
	if state == nil {
		state = &referenceRoots{}
	}
	if err := state.rebind(m); err != nil {
		return err
	}
	m.referenceMu.Lock()
	if m.referenceStates == nil {
		m.referenceStates = make(map[referenceObject]*referenceRoots)
	}
	m.referenceStates[object] = state
	m.referenceMu.Unlock()
	return nil
}

func (m *Module) requireMutable() error {
	if m == nil || !m.life.alive() {
		return ErrClosed
	}
	if !m.life.isOwned() {
		return ErrOwnership
	}
	return nil
}

// Name returns the module's registered name.
func (m *Module) Name() string {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return goString(C.WasmEdge_ModuleInstanceGetModuleName(m.ptr))
}

// HostData returns the Go value attached via NewModuleWithData, or nil.
func (m *Module) HostData() any {
	p := m.hostDataToken()
	if p == nil {
		return nil
	}
	raw, ok := moduleDataTokens.Load(p)
	if !ok {
		return nil
	}
	state, ok := raw.(*moduleDataState)
	if !ok {
		return nil
	}
	value, ok := state.value()
	if !ok {
		return nil
	}
	return value
}

func (m *Module) hostDataToken() unsafe.Pointer {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return C.WasmEdge_ModuleInstanceGetHostData(m.ptr)
}

// The C Add* functions return void in the released 0.17 API (a duplicate
// name silently replaces nothing — the engine keeps the first entry), but
// they return WasmEdge_Result at upstream HEAD. The Go signatures return
// error now so the 0.18 upgrade is not a breaking change here; today the
// error only reports Go-side misuse (see PLAN.md Phase 9).

// AddFunction moves a host function into the module under name. On success
// the module owns the function: its wrapper becomes inert (Close is a
// no-op) and the Go closure lives until the module is closed.
func (m *Module) AddFunction(name string, f *Function) error {
	defer runtime.KeepAlive(m)
	if err := m.requireMutable(); err != nil {
		return fmt.Errorf("wasmedge: add function to module: %w", err)
	}
	if f == nil {
		return fmt.Errorf("wasmedge: add function %q: %w", name, ErrInvalidArgument)
	}
	if _, exists := m.Function(name); exists {
		return fmt.Errorf("wasmedge: add function %q: %w", name, ErrAlreadyExists)
	}
	ptr, token := f.ptr, f.token
	if err := f.life.transfer(); err != nil {
		return fmt.Errorf("wasmedge: add function %q: %w", name, err)
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddFunction(m.ptr, cname, ptr)
	if token != nil {
		if entry, ok := hostFuncEntryForToken(token); ok {
			entry.module.Store(uintptr(unsafe.Pointer(m.ptr)))
		}
		m.adopted = append(m.adopted, token)
	}
	runtime.KeepAlive(f)
	return nil
}

// AddTable moves a table instance into the module under name.
func (m *Module) AddTable(name string, t *Table) error {
	defer runtime.KeepAlive(m)
	if err := m.requireMutable(); err != nil {
		return fmt.Errorf("wasmedge: add table to module: %w", err)
	}
	if t == nil {
		return fmt.Errorf("wasmedge: add table %q: %w", name, ErrInvalidArgument)
	}
	if _, exists := m.Table(name); exists {
		return fmt.Errorf("wasmedge: add table %q: %w", name, ErrAlreadyExists)
	}
	if err := t.life.transferable(); err != nil {
		return fmt.Errorf("wasmedge: add table %q: %w", name, err)
	}
	ptr := t.ptr
	object := referenceObject{kind: referenceObjectTable, ptr: uintptr(unsafe.Pointer(ptr))}
	if err := m.adoptReferenceState(object, t.roots); err != nil {
		return fmt.Errorf("wasmedge: add table %q references: %w", name, err)
	}
	if err := t.life.transfer(); err != nil {
		m.referenceMu.Lock()
		delete(m.referenceStates, object)
		m.referenceMu.Unlock()
		_ = t.roots.rebind(t)
		return fmt.Errorf("wasmedge: add table %q: %w", name, err)
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddTable(m.ptr, cname, ptr)
	t.roots = nil
	runtime.KeepAlive(t)
	return nil
}

// AddMemory moves a memory instance into the module under name.
func (m *Module) AddMemory(name string, mem *Memory) error {
	defer runtime.KeepAlive(m)
	if err := m.requireMutable(); err != nil {
		return fmt.Errorf("wasmedge: add memory to module: %w", err)
	}
	if mem == nil {
		return fmt.Errorf("wasmedge: add memory %q: %w", name, ErrInvalidArgument)
	}
	if _, exists := m.Memory(name); exists {
		return fmt.Errorf("wasmedge: add memory %q: %w", name, ErrAlreadyExists)
	}
	ptr := mem.ptr
	if err := mem.life.transfer(); err != nil {
		return fmt.Errorf("wasmedge: add memory %q: %w", name, err)
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddMemory(m.ptr, cname, ptr)
	runtime.KeepAlive(mem)
	return nil
}

// AddGlobal moves a global instance into the module under name.
func (m *Module) AddGlobal(name string, g *Global) error {
	defer runtime.KeepAlive(m)
	if err := m.requireMutable(); err != nil {
		return fmt.Errorf("wasmedge: add global to module: %w", err)
	}
	if g == nil {
		return fmt.Errorf("wasmedge: add global %q: %w", name, ErrInvalidArgument)
	}
	if _, exists := m.Global(name); exists {
		return fmt.Errorf("wasmedge: add global %q: %w", name, ErrAlreadyExists)
	}
	if err := g.life.transferable(); err != nil {
		return fmt.Errorf("wasmedge: add global %q: %w", name, err)
	}
	ptr := g.ptr
	object := referenceObject{kind: referenceObjectGlobal, ptr: uintptr(unsafe.Pointer(ptr))}
	if err := m.adoptReferenceState(object, g.roots); err != nil {
		return fmt.Errorf("wasmedge: add global %q references: %w", name, err)
	}
	if err := g.life.transfer(); err != nil {
		m.referenceMu.Lock()
		delete(m.referenceStates, object)
		m.referenceMu.Unlock()
		_ = g.roots.rebind(g)
		return fmt.Errorf("wasmedge: add global %q: %w", name, err)
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddGlobal(m.ptr, cname, ptr)
	g.roots = nil
	runtime.KeepAlive(g)
	return nil
}

// Function looks up an exported function. The borrowed view is valid while
// the module and its owner generation remain valid. The second result is
// false when no such export exists.
func (m *Module) Function(name string) (*Function, bool) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	f := borrowedFunction(C.WasmEdge_ModuleInstanceFindFunction(m.ptr, cname), m)
	return f, f != nil
}

// Memory looks up an exported memory (borrowed and generation-bounded with
// its module).
func (m *Module) Memory(name string) (*Memory, bool) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	mem := borrowedMemory(C.WasmEdge_ModuleInstanceFindMemory(m.ptr, cname), m)
	return mem, mem != nil
}

// Table looks up an exported table (borrowed and generation-bounded with its
// module).
func (m *Module) Table(name string) (*Table, bool) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	t := borrowedTable(C.WasmEdge_ModuleInstanceFindTable(m.ptr, cname), m)
	return t, t != nil
}

// Global looks up an exported global (borrowed and generation-bounded with
// its module).
func (m *Module) Global(name string) (*Global, bool) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	g := borrowedGlobal(C.WasmEdge_ModuleInstanceFindGlobal(m.ptr, cname), m)
	return g, g != nil
}

// Tag looks up an exported exception tag (borrowed and generation-bounded
// with its module).
func (m *Module) Tag(name string) (*Tag, bool) {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	t := borrowedTag(C.WasmEdge_ModuleInstanceFindTag(m.ptr, cname), m)
	return t, t != nil
}

// FunctionNames lists the module's exported function names.
func (m *Module) FunctionNames() []string {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListFunctionLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListFunction(m.ptr, buf, n)
		})
}

// MemoryNames lists the module's exported memory names.
func (m *Module) MemoryNames() []string {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListMemoryLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListMemory(m.ptr, buf, n)
		})
}

// TableNames lists the module's exported table names.
func (m *Module) TableNames() []string {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListTableLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListTable(m.ptr, buf, n)
		})
}

// GlobalNames lists the module's exported global names.
func (m *Module) GlobalNames() []string {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListGlobalLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListGlobal(m.ptr, buf, n)
		})
}

// TagNames lists the module's exported exception-tag names.
func (m *Module) TagNames() []string {
	m.assertAlive()
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListTagLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListTag(m.ptr, buf, n)
		})
}

// Close destroys an owned module instance and releases the Go closures of
// its adopted host functions. No-op for borrowed views and repeat calls.
func (m *Module) Close() error {
	ptr := m.ptr
	adopted := m.adopted
	wasi := m.wasi
	owned := m.life.ownedAndOpen()
	err := m.life.close(func() {
		detached := make([]unsafe.Pointer, 0, len(adopted))
		for _, token := range adopted {
			if unpublishHostFuncToken(token) {
				detached = append(detached, token)
			}
		}
		C.WasmEdge_ModuleInstanceDelete(ptr)
		for _, token := range detached {
			deleteOpaqueToken(token)
		}
		if wasi != nil {
			wasi.close()
		}
		m.roots.close()
		m.referenceMu.Lock()
		states := m.referenceStates
		m.referenceStates = nil
		m.referenceMu.Unlock()
		for _, state := range states {
			state.close()
		}
		for i := len(m.dependencies) - 1; i >= 0; i-- {
			m.dependencies[i]()
		}
		m.dependencies = nil
		m.dependencyModules = nil
		for _, state := range m.registrations {
			state.forget([]*Module{m})
		}
		m.registrations = nil
	})
	if err == nil {
		m.adopted = nil
		if owned {
			m.wasi = nil
		}
	}
	return err
}
