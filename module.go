package wasmedge

// #include "shims.h"
import "C"

import (
	"runtime"
	"runtime/cgo"
)

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
	adopted []cgo.Handle
}

// NewModule creates an empty named host module. Fill it with AddFunction
// (and friends), then register it into a VM, Executor or Store.
func NewModule(name string) *Module {
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.WasmEdge_ModuleInstanceCreate(cname)
	if ptr == nil {
		return nil
	}
	return ownedModule(ptr)
}

// NewModuleWithData creates a named host module carrying arbitrary Go host
// data, retrievable via HostData from any borrowed view of the module. The
// pin on data is released when the module instance is destroyed, wherever
// that happens (Close here, or inside the engine).
func NewModuleWithData(name string, data any) *Module {
	cname := newWEString(name)
	defer freeWEString(cname)
	h := cgo.NewHandle(data)
	ptr := C.wasmedgego_moduleCreateWithData(cname, C.uintptr_t(h))
	if ptr == nil {
		h.Delete()
		return nil
	}
	return ownedModule(ptr)
}

func ownedModule(ptr *C.WasmEdge_ModuleInstanceContext) *Module {
	m := &Module{ptr: ptr}
	// The cleanup frees only the C object; adopted host-function handles
	// are deliberately left to the explicit Close path — a leaked module
	// wrapper leaks its Go closures (reported under wasmedge_debug) rather
	// than risking a use-after-free from a concurrently running call.
	arm(m, &m.life, "Module", func() { C.WasmEdge_ModuleInstanceDelete(ptr) })
	return m
}

func borrowedModule(ptr *C.WasmEdge_ModuleInstanceContext, owner any) *Module {
	if ptr == nil {
		return nil
	}
	return &Module{ptr: ptr, life: borrowed(owner)}
}

// Name returns the module's registered name.
func (m *Module) Name() string {
	defer runtime.KeepAlive(m)
	return goString(C.WasmEdge_ModuleInstanceGetModuleName(m.ptr))
}

// HostData returns the Go value attached via NewModuleWithData, or nil.
func (m *Module) HostData() any {
	defer runtime.KeepAlive(m)
	p := C.WasmEdge_ModuleInstanceGetHostData(m.ptr)
	if p == nil {
		return nil
	}
	return cgo.Handle(uintptr(p)).Value()
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
	if f == nil || !f.life.alive() {
		return &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "adding a nil or closed function"}
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddFunction(m.ptr, cname, f.ptr)
	if f.life.transfer() && f.handle != 0 {
		m.adopted = append(m.adopted, f.handle)
	}
	runtime.KeepAlive(f)
	return nil
}

// AddTable moves a table instance into the module under name.
func (m *Module) AddTable(name string, t *Table) error {
	defer runtime.KeepAlive(m)
	if t == nil || !t.life.alive() {
		return &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "adding a nil or closed table"}
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddTable(m.ptr, cname, t.ptr)
	t.life.transfer()
	runtime.KeepAlive(t)
	return nil
}

// AddMemory moves a memory instance into the module under name.
func (m *Module) AddMemory(name string, mem *Memory) error {
	defer runtime.KeepAlive(m)
	if mem == nil || !mem.life.alive() {
		return &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "adding a nil or closed memory"}
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddMemory(m.ptr, cname, mem.ptr)
	mem.life.transfer()
	runtime.KeepAlive(mem)
	return nil
}

// AddGlobal moves a global instance into the module under name.
func (m *Module) AddGlobal(name string, g *Global) error {
	defer runtime.KeepAlive(m)
	if g == nil || !g.life.alive() {
		return &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "adding a nil or closed global"}
	}
	cname := newWEString(name)
	defer freeWEString(cname)
	C.WasmEdge_ModuleInstanceAddGlobal(m.ptr, cname, g.ptr)
	g.life.transfer()
	runtime.KeepAlive(g)
	return nil
}

// Function looks up an exported function (borrowed; valid while the module
// lives). The second result is false when no such export exists.
func (m *Module) Function(name string) (*Function, bool) {
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	f := borrowedFunction(C.WasmEdge_ModuleInstanceFindFunction(m.ptr, cname), m)
	return f, f != nil
}

// Memory looks up an exported memory (borrowed).
func (m *Module) Memory(name string) (*Memory, bool) {
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	mem := borrowedMemory(C.WasmEdge_ModuleInstanceFindMemory(m.ptr, cname), m)
	return mem, mem != nil
}

// Table looks up an exported table (borrowed).
func (m *Module) Table(name string) (*Table, bool) {
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	t := borrowedTable(C.WasmEdge_ModuleInstanceFindTable(m.ptr, cname), m)
	return t, t != nil
}

// Global looks up an exported global (borrowed).
func (m *Module) Global(name string) (*Global, bool) {
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	g := borrowedGlobal(C.WasmEdge_ModuleInstanceFindGlobal(m.ptr, cname), m)
	return g, g != nil
}

// Tag looks up an exported exception tag (borrowed).
func (m *Module) Tag(name string) (*Tag, bool) {
	defer runtime.KeepAlive(m)
	cname := newWEString(name)
	defer freeWEString(cname)
	t := borrowedTag(C.WasmEdge_ModuleInstanceFindTag(m.ptr, cname), m)
	return t, t != nil
}

// FunctionNames lists the module's exported function names.
func (m *Module) FunctionNames() []string {
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListFunctionLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListFunction(m.ptr, buf, n)
		})
}

// MemoryNames lists the module's exported memory names.
func (m *Module) MemoryNames() []string {
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListMemoryLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListMemory(m.ptr, buf, n)
		})
}

// TableNames lists the module's exported table names.
func (m *Module) TableNames() []string {
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListTableLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListTable(m.ptr, buf, n)
		})
}

// GlobalNames lists the module's exported global names.
func (m *Module) GlobalNames() []string {
	defer runtime.KeepAlive(m)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_ModuleInstanceListGlobalLength(m.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_ModuleInstanceListGlobal(m.ptr, buf, n)
		})
}

// TagNames lists the module's exported exception-tag names.
func (m *Module) TagNames() []string {
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
	err := m.life.close(func() {
		C.WasmEdge_ModuleInstanceDelete(ptr)
		for _, h := range adopted {
			h.Delete()
		}
	})
	if err == nil {
		m.adopted = nil
	}
	return err
}
