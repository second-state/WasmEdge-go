package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"context"
	"runtime"
	"unsafe"
)

// VM is the high-level embedding entry point: it bundles a loader,
// validator, executor, store and statistics behind load/validate/
// instantiate/execute workflows.
type VM struct {
	ptr  *C.WasmEdge_VMContext
	life lifetime
}

// VMOption configures NewVM.
type VMOption func(*vmOptions)

type vmOptions struct {
	store *Store
}

// WithExternalStore makes the VM operate on a caller-owned Store (which
// must outlive the VM) instead of an internal one.
func WithExternalStore(s *Store) VMOption {
	return func(o *vmOptions) { o.store = s }
}

// NewVM creates a VM honoring cfg (nil for defaults).
func NewVM(cfg *Config, opts ...VMOption) (*VM, error) {
	var o vmOptions
	for _, opt := range opts {
		opt(&o)
	}
	ccfg, free := cfg.build()
	defer free()
	var cstore *C.WasmEdge_StoreContext
	if o.store != nil {
		cstore = o.store.ptr
	}
	ptr := C.WasmEdge_VMCreate(ccfg, cstore)
	runtime.KeepAlive(o.store)
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError, Message: "VM creation failed"}
	}
	vm := &VM{ptr: ptr}
	arm(vm, &vm.life, "VM", func() { C.WasmEdge_VMDelete(ptr) })
	return vm, nil
}

// ---- registration ----------------------------------------------------------

// RegisterModule instantiates ast into the VM's store under name, making
// its exports importable and callable via ExecuteRegistered. The ASTModule
// remains caller-owned.
func (vm *VM) RegisterModule(name string, ast *ASTModule) error {
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(ast)
	cname := newWEString(name)
	defer freeWEString(cname)
	return newResult(C.WasmEdge_VMRegisterModuleFromASTModule(vm.ptr, cname, ast.ptr))
}

// RegisterModuleBytes loads and registers a module from a WASM binary
// under name.
func (vm *VM) RegisterModuleBytes(name string, b []byte) error {
	defer runtime.KeepAlive(vm)
	cname := newWEString(name)
	defer freeWEString(cname)
	err := newResult(C.WasmEdge_VMRegisterModuleFromBytes(vm.ptr, cname, wrapBytes(b)))
	runtime.KeepAlive(b)
	return err
}

// RegisterImport registers a host (or instantiated) module under its own
// name. The module remains caller-owned and must outlive the VM's use of
// it.
func (vm *VM) RegisterImport(mod *Module) error {
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(mod)
	return newResult(C.WasmEdge_VMRegisterModuleFromImport(vm.ptr, mod.ptr))
}

// TODO(intern-easy): A4 — bind the remaining VM conveniences:
//
//	RegisterModuleFile(name, path string) error
//	    -> WasmEdge_VMRegisterModuleFromFile (pattern: RegisterModuleBytes
//	       + LoadFile's C.CString handling in loader.go)
//	RegisterImportWithAlias(mod *Module, alias string) error
//	    -> WasmEdge_VMRegisterModuleFromImportWithAlias
//	RunFile(path, fn string, params ...Value) ([]Value, error)
//	    -> staged like RunBytes below, via LoadFile
//	ExecuteRegistered(mod, fn string, params ...Value) ([]Value, error)
//	    -> WasmEdge_VMExecuteRegistered; size returns via
//	       WasmEdge_VMGetFunctionTypeRegistered (pattern: Execute below)
//	DeleteRegisteredModule(name string)
//	    -> WasmEdge_VMDeleteRegisteredModule
//
// Extend TestVMRegisterModule in vm_test.go for each addition.

// ---- staged workflow -------------------------------------------------------

// LoadBytes starts the staged workflow: load a module from a WASM binary.
// Follow with Validate, Instantiate, Execute.
func (vm *VM) LoadBytes(b []byte) error {
	defer runtime.KeepAlive(vm)
	err := newResult(C.WasmEdge_VMLoadWasmFromBytes(vm.ptr, wrapBytes(b)))
	runtime.KeepAlive(b)
	return err
}

// LoadFile starts the staged workflow from a file path.
func (vm *VM) LoadFile(path string) error {
	defer runtime.KeepAlive(vm)
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return newResult(C.WasmEdge_VMLoadWasmFromFile(vm.ptr, cpath))
}

// Load starts the staged workflow from an already loaded ASTModule (which
// remains caller-owned).
func (vm *VM) Load(ast *ASTModule) error {
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(ast)
	return newResult(C.WasmEdge_VMLoadWasmFromASTModule(vm.ptr, ast.ptr))
}

// Validate checks the loaded module; must follow a Load*.
func (vm *VM) Validate() error {
	defer runtime.KeepAlive(vm)
	return newResult(C.WasmEdge_VMValidate(vm.ptr))
}

// Instantiate instantiates the validated module as the VM's active module;
// must follow Validate.
func (vm *VM) Instantiate() error {
	defer runtime.KeepAlive(vm)
	return newResult(C.WasmEdge_VMInstantiate(vm.ptr))
}

// Execute invokes an exported function of the active module.
func (vm *VM) Execute(fn string, params ...Value) ([]Value, error) {
	defer runtime.KeepAlive(vm)
	cname := newWEString(fn)
	defer freeWEString(cname)

	ft := borrowedFunctionType(C.WasmEdge_VMGetFunctionType(vm.ptr, cname))
	if ft == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "wasm function not found: " + fn}
	}
	nret := len(ft.Results())
	cparams := packValues(params)
	creturns := make([]C.WasmEdge_Value, max(nret, 1))
	err := newResult(C.WasmEdge_VMExecute(vm.ptr, cname,
		valuesPtr(cparams), C.uint32_t(len(cparams)),
		&creturns[0], C.uint32_t(nret)))
	if err != nil {
		return nil, err
	}
	return unpackValues(creturns[:nret]), nil
}

// ---- one-shot --------------------------------------------------------------

// RunBytes loads, validates, instantiates and executes in one call,
// replacing the VM's active module. Implemented as the staged sequence
// (rather than the C one-shot API) so the return arity comes from the
// function type instead of a caller-guessed buffer size.
func (vm *VM) RunBytes(b []byte, fn string, params ...Value) ([]Value, error) {
	if err := vm.LoadBytes(b); err != nil {
		return nil, err
	}
	if err := vm.Validate(); err != nil {
		return nil, err
	}
	if err := vm.Instantiate(); err != nil {
		return nil, err
	}
	return vm.Execute(fn, params...)
}

// Run loads from an ASTModule and executes like RunBytes.
func (vm *VM) Run(ast *ASTModule, fn string, params ...Value) ([]Value, error) {
	if err := vm.Load(ast); err != nil {
		return nil, err
	}
	if err := vm.Validate(); err != nil {
		return nil, err
	}
	if err := vm.Instantiate(); err != nil {
		return nil, err
	}
	return vm.Execute(fn, params...)
}

// ---- async -----------------------------------------------------------------

// ExecuteAsync starts an asynchronous invocation of an exported function
// of the active module.
func (vm *VM) ExecuteAsync(fn string, params ...Value) (*Execution, error) {
	defer runtime.KeepAlive(vm)
	cname := newWEString(fn)
	defer freeWEString(cname)
	cparams := packValues(params)
	ptr := C.WasmEdge_VMAsyncExecute(vm.ptr, cname,
		valuesPtr(cparams), C.uint32_t(len(cparams)))
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "async execution failed to start"}
	}
	return newExecution(ptr), nil
}

// ExecuteContext is Execute with cancellation: when ctx fires, the WASM
// execution is interrupted and ctx's error is returned.
func (vm *VM) ExecuteContext(ctx context.Context, fn string, params ...Value) ([]Value, error) {
	ex, err := vm.ExecuteAsync(fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, ex)
}

// ---- introspection ---------------------------------------------------------

// VMFunction describes one exported function of the active module.
type VMFunction struct {
	Name string
	Type *FunctionType // borrowed; valid while the VM lives
}

// Functions lists the exported functions of the active module.
func (vm *VM) Functions() []VMFunction {
	defer runtime.KeepAlive(vm)
	n := C.WasmEdge_VMGetFunctionListLength(vm.ptr)
	if n == 0 {
		return nil
	}
	names := make([]C.WasmEdge_String, n)
	types := make([]*C.WasmEdge_FunctionTypeContext, n)
	got := C.WasmEdge_VMGetFunctionList(vm.ptr, &names[0], &types[0], n)
	out := make([]VMFunction, 0, got)
	for i := range int(min(got, n)) {
		out = append(out, VMFunction{
			Name: goString(names[i]),
			Type: borrowedFunctionType(types[i]),
		})
	}
	return out
}

// FunctionType returns the type of an exported function of the active
// module (borrowed), with ok=false when the export does not exist.
func (vm *VM) FunctionType(fn string) (*FunctionType, bool) {
	defer runtime.KeepAlive(vm)
	cname := newWEString(fn)
	defer freeWEString(cname)
	ft := borrowedFunctionType(C.WasmEdge_VMGetFunctionType(vm.ptr, cname))
	return ft, ft != nil
}

// ActiveModule returns the anonymous active module instance (borrowed), or
// nil before Instantiate.
func (vm *VM) ActiveModule() *Module {
	defer runtime.KeepAlive(vm)
	return borrowedModule(C.WasmEdge_VMGetActiveModule(vm.ptr))
}

// RegisteredModule looks up a module registered into this VM (borrowed).
func (vm *VM) RegisteredModule(name string) (*Module, bool) {
	defer runtime.KeepAlive(vm)
	cname := newWEString(name)
	defer freeWEString(cname)
	m := borrowedModule(C.WasmEdge_VMGetRegisteredModule(vm.ptr, cname))
	return m, m != nil
}

// RegisteredModuleNames lists modules registered into this VM.
func (vm *VM) RegisteredModuleNames() []string {
	defer runtime.KeepAlive(vm)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_VMListRegisteredModuleLength(vm.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_VMListRegisteredModule(vm.ptr, buf, n)
		})
}

// Store returns the VM's store (borrowed).
func (vm *VM) Store() *Store {
	defer runtime.KeepAlive(vm)
	return &Store{ptr: C.WasmEdge_VMGetStoreContext(vm.ptr), life: borrowed()}
}

// Stats returns the VM's statistics collector (borrowed).
func (vm *VM) Stats() *Statistics {
	defer runtime.KeepAlive(vm)
	return borrowedStatistics(C.WasmEdge_VMGetStatisticsContext(vm.ptr))
}

// Reset clears the VM back to the freshly created state (the C API's
// VMCleanup): the active module, loaded module and registered modules are
// dropped.
func (vm *VM) Reset() {
	defer runtime.KeepAlive(vm)
	C.WasmEdge_VMCleanup(vm.ptr)
}

// Close destroys the VM (and everything it owns: internal store, loader,
// validator, executor, statistics, registered non-import modules). No-op
// after the first call.
func (vm *VM) Close() error {
	ptr := vm.ptr
	return vm.life.close(func() { C.WasmEdge_VMDelete(ptr) })
}
