package wasmedge

// #include <stdlib.h>
// #include <wasmedge/wasmedge.h>
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"unsafe"
)

// VM is the high-level embedding entry point: it bundles a loader,
// validator, executor, store and statistics behind load/validate/
// instantiate/execute workflows.
type VM struct {
	ptr            *C.WasmEdge_VMContext
	life           lifetime
	gate           executionGate
	store          *Store
	storeState     *storeState
	releaseStore   func()
	imports        []*Module
	importReleases []func()
	dependencies   map[*Module]func()
	// ownedStoreModules are registrations whose module instances are owned
	// and deleted by the VM. Caller-owned imports survive VM.Close when an
	// external Store is used and therefore remain in that Store's state.
	ownedStoreModules []*Module
	active            generation
	registered        generation
}

// VMOption configures NewVM.
type VMOption func(*vmOptions)

type vmOptions struct {
	store *Store
}

// WithExternalStore makes the VM operate on a caller-owned Store instead of
// an internal one. The VM leases it until Close, so an early Store.Close
// returns ErrInUse. WasmEdge_VMCleanup clears every non-built-in registration
// from that store; consequently VM.Reset also clears modules that were
// registered in the external store before the VM was created.
func WithExternalStore(s *Store) VMOption {
	return func(o *vmOptions) { o.store = s }
}

// NewVM creates a VM honoring cfg (nil for defaults).
func NewVM(cfg *Config, opts ...VMOption) (*VM, error) {
	var o vmOptions
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("VM option %d is nil: %w", i, ErrInvalidArgument)
		}
		opt(&o)
	}
	ccfg, free, err := cfg.build()
	if err != nil {
		return nil, fmt.Errorf("create VM configuration: %w", err)
	}
	defer free()

	var cstore *C.WasmEdge_StoreContext
	releaseStore := func() {}
	if o.store != nil {
		releaseStore, err = o.store.acquireLease()
		if err != nil {
			return nil, fmt.Errorf("attach external store: %w", err)
		}
		cstore = o.store.ptr
	}
	ptr := C.WasmEdge_VMCreate(ccfg, cstore)
	runtime.KeepAlive(o.store)
	if ptr == nil {
		releaseStore()
		return nil, fmt.Errorf("create VM: %w", ErrUnavailable)
	}
	storeState := &storeState{}
	if o.store != nil {
		storeState = o.store.state
	}
	vm := &VM{
		ptr: ptr, store: o.store, storeState: storeState,
		releaseStore: releaseStore,
	}
	arm(vm, &vm.life, "VM", func() { C.WasmEdge_VMDelete(ptr) })
	return vm, nil
}

func (vm *VM) assertAlive() { vm.life.assertAlive("VM") }

func (vm *VM) acquireLease() (func(), error) {
	return vm.life.acquire("VM")
}

func (vm *VM) retainReference(root any) bool {
	vm.assertAlive()
	return vm.registered.retain(vm, root) == nil
}

func (vm *VM) ensureUnused() error {
	if err := vm.life.ensureUnused(); err != nil {
		return fmt.Errorf("VM operation: %w", err)
	}
	return nil
}

func (vm *VM) activeGuard() *generationGuard {
	return guardGeneration(vm, &vm.active)
}

func (vm *VM) registeredGuard() *generationGuard {
	return guardGeneration(vm, &vm.registered)
}

func (vm *VM) registeredOwner(ptr *C.WasmEdge_ModuleInstanceContext) any {
	for i := len(vm.storeState.retained) - 1; i >= 0; i-- {
		module := vm.storeState.retained[i]
		if module != nil && module.ptr == ptr {
			return module
		}
	}
	return vm.registeredGuard()
}

func (vm *VM) bindContext(ctx context.Context) (func(), error) {
	if ctx == nil {
		return func() {}, nil
	}
	executor := C.WasmEdge_VMGetExecutorContext(vm.ptr)
	if executor == nil {
		return nil, ErrUnavailable
	}
	return bindExecutionContext(unsafe.Pointer(executor), ctx)
}

func (vm *VM) prepareStoreDependencies() (func(bool), error) {
	prepared := make(map[*Module]func())
	for _, module := range vm.storeState.retained {
		if module == nil || ownerChainContains(module, vm) {
			continue
		}
		if _, exists := vm.dependencies[module]; exists {
			continue
		}
		if _, exists := prepared[module]; exists {
			continue
		}
		release, err := module.acquireLease()
		if err != nil {
			for _, rollback := range prepared {
				rollback()
			}
			return nil, err
		}
		prepared[module] = release
	}
	var finished bool
	return func(commit bool) {
		if finished {
			return
		}
		finished = true
		if !commit {
			for _, release := range prepared {
				release()
			}
			return
		}
		if vm.dependencies == nil {
			vm.dependencies = make(map[*Module]func())
		}
		for module, release := range prepared {
			vm.dependencies[module] = release
		}
	}, nil
}

func (vm *VM) releaseStoreDependencies() {
	for module, release := range vm.dependencies {
		release()
		delete(vm.dependencies, module)
	}
	vm.dependencies = nil
}

// ---- registration ----------------------------------------------------------

// RegisterModule instantiates ast into the VM's store under name, making
// its exports importable and callable via ExecuteRegistered. The ASTModule
// remains caller-owned.
func (vm *VM) RegisterModule(name string, ast *ASTModule) error {
	vm.assertAlive()
	if ast == nil {
		return fmt.Errorf("register AST module requires a non-nil module: %w", ErrInvalidArgument)
	}
	ast.assertAlive()
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	finishDependencies, err := vm.prepareStoreDependencies()
	if err != nil {
		return fmt.Errorf("register module dependencies: %w", err)
	}
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(ast)
	cname := newWEString(name)
	defer freeWEString(cname)
	err = newResult(C.WasmEdge_VMRegisterModuleFromASTModule(vm.ptr, cname, ast.ptr))
	finishDependencies(err == nil)
	if err == nil {
		vm.retainStoreModule(name, nil)
	}
	return err
}

// RegisterModuleBytes loads and registers a module from a WASM binary
// under name.
func (vm *VM) RegisterModuleBytes(name string, b []byte) error {
	vm.assertAlive()
	bytes, err := wrapBytes(b)
	if err != nil {
		return err
	}
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	finishDependencies, err := vm.prepareStoreDependencies()
	if err != nil {
		return fmt.Errorf("register bytecode module dependencies: %w", err)
	}
	defer runtime.KeepAlive(vm)
	cname := newWEString(name)
	defer freeWEString(cname)
	err = newResult(C.WasmEdge_VMRegisterModuleFromBytes(vm.ptr, cname, bytes))
	runtime.KeepAlive(b)
	finishDependencies(err == nil)
	if err == nil {
		vm.retainStoreModule(name, nil)
	}
	return err
}

// RegisterModuleFile loads and registers a module from path under name.
func (vm *VM) RegisterModuleFile(name, path string) error {
	vm.assertAlive()
	if err := validateCString("WASM path", path); err != nil {
		return err
	}
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	finishDependencies, err := vm.prepareStoreDependencies()
	if err != nil {
		return fmt.Errorf("register file module dependencies: %w", err)
	}
	defer runtime.KeepAlive(vm)
	cname := newWEString(name)
	defer freeWEString(cname)
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	err = newResult(C.WasmEdge_VMRegisterModuleFromFile(vm.ptr, cname, cpath))
	finishDependencies(err == nil)
	if err == nil {
		vm.retainStoreModule(name, nil)
	}
	return err
}

// RegisterImport registers a host (or instantiated) module under its own
// name. The module remains caller-owned; the VM leases it until Reset or
// Close, so an early Module.Close returns ErrInUse. WithExternalStore, the
// registration itself survives VM.Close as a direct Store registration and
// is removed when the caller closes the module. Borrowed views are rejected.
func (vm *VM) RegisterImport(mod *Module) error {
	vm.assertAlive()
	if mod == nil {
		return fmt.Errorf("register import requires a non-nil module: %w", ErrInvalidArgument)
	}
	mod.assertAlive()
	if !mod.life.isOwned() {
		return fmt.Errorf("register import requires a caller-owned module: %w", ErrOwnership)
	}
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	release, err := mod.acquireLease()
	if err != nil {
		return fmt.Errorf("register import: %w", err)
	}
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(mod)
	err = newResult(C.WasmEdge_VMRegisterModuleFromImport(vm.ptr, mod.ptr))
	if err == nil {
		vm.imports = append(vm.imports, mod)
		vm.importReleases = append(vm.importReleases, release)
		vm.retainStoreModule(mod.Name(), mod)
	} else {
		release()
	}
	return err
}

// RegisterImportWithAlias registers mod under alias instead of the module's
// own name. The VM leases it until Reset or Close. WithExternalStore, the
// alias remains a direct Store registration after Close until mod is closed.
// Borrowed views are rejected.
func (vm *VM) RegisterImportWithAlias(mod *Module, alias string) error {
	vm.assertAlive()
	if mod == nil {
		return fmt.Errorf("register import alias requires a non-nil module: %w", ErrInvalidArgument)
	}
	mod.assertAlive()
	if !mod.life.isOwned() {
		return fmt.Errorf("register import alias requires a caller-owned module: %w", ErrOwnership)
	}
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	release, err := mod.acquireLease()
	if err != nil {
		return fmt.Errorf("register import alias: %w", err)
	}
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(mod)
	calias := newWEString(alias)
	defer freeWEString(calias)
	err = newResult(C.WasmEdge_VMRegisterModuleFromImportWithAlias(
		vm.ptr, calias, mod.ptr))
	if err == nil {
		vm.imports = append(vm.imports, mod)
		vm.importReleases = append(vm.importReleases, release)
		vm.retainStoreModule(alias, mod)
	} else {
		release()
	}
	return err
}

func (vm *VM) retainStoreModule(name string, existing *Module) {
	ownedByVM := existing == nil
	module := existing
	if module == nil {
		cname := newWEString(name)
		defer freeWEString(cname)
		ptr := C.WasmEdge_VMGetRegisteredModule(vm.ptr, cname)
		module = borrowedModule(ptr, vm.registeredGuard())
	}
	if module != nil {
		vm.storeState.retain(module)
		if ownedByVM {
			vm.ownedStoreModules = append(vm.ownedStoreModules, module)
		}
	}
}

// ---- staged workflow -------------------------------------------------------

// LoadBytes starts the staged workflow: load a module from a WASM binary.
// Follow with Validate, Instantiate, Execute.
func (vm *VM) LoadBytes(b []byte) error {
	vm.assertAlive()
	bytes, err := wrapBytes(b)
	if err != nil {
		return err
	}
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	defer runtime.KeepAlive(vm)
	err = newResult(C.WasmEdge_VMLoadWasmFromBytes(vm.ptr, bytes))
	runtime.KeepAlive(b)
	return err
}

// LoadFile starts the staged workflow from a file path.
func (vm *VM) LoadFile(path string) error {
	vm.assertAlive()
	if err := validateCString("WASM path", path); err != nil {
		return err
	}
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	defer runtime.KeepAlive(vm)
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return newResult(C.WasmEdge_VMLoadWasmFromFile(vm.ptr, cpath))
}

// Load starts the staged workflow from an already loaded ASTModule (which
// remains caller-owned).
func (vm *VM) Load(ast *ASTModule) error {
	vm.assertAlive()
	if ast == nil {
		return fmt.Errorf("load requires a non-nil AST module: %w", ErrInvalidArgument)
	}
	ast.assertAlive()
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(ast)
	return newResult(C.WasmEdge_VMLoadWasmFromASTModule(vm.ptr, ast.ptr))
}

// Validate checks the loaded module; must follow a Load*.
func (vm *VM) Validate() error {
	vm.assertAlive()
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	defer runtime.KeepAlive(vm)
	return newResult(C.WasmEdge_VMValidate(vm.ptr))
}

// Instantiate instantiates the validated module as the VM's active module;
// must follow Validate.
func (vm *VM) Instantiate() error {
	vm.assertAlive()
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	finishDependencies, err := vm.prepareStoreDependencies()
	if err != nil {
		return fmt.Errorf("instantiate module dependencies: %w", err)
	}
	defer runtime.KeepAlive(vm)
	err = newResult(C.WasmEdge_VMInstantiate(vm.ptr))
	finishDependencies(err == nil)
	if err == nil {
		vm.active.advance()
	}
	return err
}

// Execute invokes an exported function of the active module.
func (vm *VM) Execute(fn string, params ...Value) ([]Value, error) {
	vm.assertAlive()
	assertValueOwnersAlive(params)
	cname := newWEString(fn)
	defer freeWEString(cname)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	releaseGate, err := vm.gate.begin()
	if err != nil {
		return nil, fmt.Errorf("execute function: %w", err)
	}
	defer releaseGate()
	owner := vm.activeGuard()
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(params)

	ft, ok := functionTypeFromC(C.WasmEdge_VMGetFunctionType(vm.ptr, cname))
	if !ok {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "wasm function not found: " + fn}
	}
	if err := validateInvocationParams(ft, params); err != nil {
		return nil, err
	}
	finishReferences, err := owner.prepareInvocationReferences(params)
	if err != nil {
		return nil, fmt.Errorf("retain invocation references: %w", err)
	}
	defer finishReferences(false)
	nret := len(ft.Results)
	creturns := make([]C.WasmEdge_Value, max(nret, 1))
	err = newResult(C.WasmEdge_VMExecute(vm.ptr, cname,
		valuesPtr(cparams), paramCount,
		&creturns[0], C.uint32_t(nret)))
	// Once native invocation begins, guest code may have persisted a
	// reference even when the eventual result is a trap.
	finishReferences(true)
	if err != nil {
		return nil, err
	}
	return unpackValuesOwned(creturns[:nret], owner, params...), nil
}

// ExecuteRegistered invokes fn from the registered module named module.
// Result storage is sized from the function type.
func (vm *VM) ExecuteRegistered(module, fn string, params ...Value) ([]Value, error) {
	vm.assertAlive()
	assertValueOwnersAlive(params)
	cmodule := newWEString(module)
	defer freeWEString(cmodule)
	cfn := newWEString(fn)
	defer freeWEString(cfn)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	releaseGate, err := vm.gate.begin()
	if err != nil {
		return nil, fmt.Errorf("execute registered function: %w", err)
	}
	defer releaseGate()
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(params)

	modulePtr := C.WasmEdge_VMGetRegisteredModule(vm.ptr, cmodule)
	if modulePtr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "registered wasm module not found: " + module}
	}
	owner := vm.registeredOwner(modulePtr)
	retainer, ok := owner.(invocationReferenceRetainer)
	if !ok {
		return nil, fmt.Errorf("retain registered invocation owner: %w", ErrOwnership)
	}
	releaseOwner, err := acquireLeases(owner)
	if err != nil {
		return nil, fmt.Errorf("lease registered module: %w", err)
	}
	defer releaseOwner()

	ft, ok := functionTypeFromC(
		C.WasmEdge_VMGetFunctionTypeRegistered(vm.ptr, cmodule, cfn))
	if !ok {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "registered wasm function not found: " + module + "." + fn}
	}
	if err := validateInvocationParams(ft, params); err != nil {
		return nil, err
	}
	finishReferences, err := retainer.prepareInvocationReferences(params)
	if err != nil {
		return nil, fmt.Errorf("retain invocation references: %w", err)
	}
	defer finishReferences(false)
	nret := len(ft.Results)
	creturns := make([]C.WasmEdge_Value, max(nret, 1))
	err = newResult(C.WasmEdge_VMExecuteRegistered(
		vm.ptr, cmodule, cfn,
		valuesPtr(cparams), paramCount,
		&creturns[0], C.uint32_t(nret)))
	finishReferences(true)
	if err != nil {
		return nil, err
	}
	return unpackValuesOwned(creturns[:nret], owner, params...), nil
}

// ---- one-shot --------------------------------------------------------------

// RunBytes atomically loads, validates, instantiates and executes in one
// native VM operation, replacing the active module.
func (vm *VM) RunBytes(b []byte, fn string, params ...Value) ([]Value, error) {
	execution, err := vm.RunBytesAsync(b, fn, params...)
	if err != nil {
		return nil, err
	}
	return execution.Wait()
}

// RunFile is the file-backed form of RunBytes.
func (vm *VM) RunFile(path, fn string, params ...Value) ([]Value, error) {
	execution, err := vm.RunFileAsync(path, fn, params...)
	if err != nil {
		return nil, err
	}
	return execution.Wait()
}

// Run is the preloaded-AST form of RunBytes.
func (vm *VM) Run(ast *ASTModule, fn string, params ...Value) ([]Value, error) {
	execution, err := vm.RunAsync(ast, fn, params...)
	if err != nil {
		return nil, err
	}
	return execution.Wait()
}

// ---- async -----------------------------------------------------------------

// RunBytesAsync starts the atomic one-shot RunBytes workflow. The bytecode is
// copied into C-owned memory because WasmEdge 0.17.1 retains its byte span on
// the detached worker thread.
func (vm *VM) RunBytesAsync(b []byte, fn string, params ...Value) (*Execution, error) {
	//nolint:staticcheck // A nil context is the internal no-cancellation sentinel.
	return vm.runBytesAsync(nil, b, fn, params...)
}

func (vm *VM) runBytesAsync(
	ctx context.Context, b []byte, fn string, params ...Value,
) (*Execution, error) {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	assertValueOwnersAlive(params)
	cfn := newWEString(fn)
	defer freeWEString(cfn)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	bytes, freeBytes, err := copyBytes(b)
	if err != nil {
		return nil, err
	}
	owner, release, finishDependencies, err := vm.beginAsyncRun(ctx, params, freeBytes)
	if err != nil {
		freeBytes()
		return nil, fmt.Errorf("run bytes asynchronously: %w", err)
	}
	ptr := C.WasmEdge_VMAsyncRunWasmFromBytes(
		vm.ptr, bytes, cfn, valuesPtr(cparams), paramCount)
	if ptr == nil {
		finishDependencies(false)
		release()
		return nil, fmt.Errorf("start async bytecode run: %w", ErrUnavailable)
	}
	finishDependencies(true)
	return newExecution(ptr, owner, params, release, vm.gate.poison), nil
}

// RunFileAsync starts the atomic file-backed one-shot workflow.
func (vm *VM) RunFileAsync(path, fn string, params ...Value) (*Execution, error) {
	//nolint:staticcheck // A nil context is the internal no-cancellation sentinel.
	return vm.runFileAsync(nil, path, fn, params...)
}

func (vm *VM) runFileAsync(
	ctx context.Context, path, fn string, params ...Value,
) (*Execution, error) {
	vm.assertAlive()
	if err := validateCString("WASM path", path); err != nil {
		return nil, err
	}
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	cfn := newWEString(fn)
	defer freeWEString(cfn)
	assertValueOwnersAlive(params)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	owner, release, finishDependencies, err := vm.beginAsyncRun(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("run file asynchronously: %w", err)
	}
	defer runtime.KeepAlive(vm)
	ptr := C.WasmEdge_VMAsyncRunWasmFromFile(
		vm.ptr, cpath, cfn, valuesPtr(cparams), paramCount)
	if ptr == nil {
		finishDependencies(false)
		release()
		return nil, fmt.Errorf("start async file run: %w", ErrUnavailable)
	}
	finishDependencies(true)
	return newExecution(ptr, owner, params, release, vm.gate.poison), nil
}

// RunAsync starts the atomic one-shot workflow from ast.
func (vm *VM) RunAsync(ast *ASTModule, fn string, params ...Value) (*Execution, error) {
	//nolint:staticcheck // A nil context is the internal no-cancellation sentinel.
	return vm.runAsync(nil, ast, fn, params...)
}

func (vm *VM) runAsync(
	ctx context.Context, ast *ASTModule, fn string, params ...Value,
) (*Execution, error) {
	vm.assertAlive()
	if ast == nil {
		return nil, fmt.Errorf("run AST module: %w", ErrInvalidArgument)
	}
	ast.assertAlive()
	cfn := newWEString(fn)
	defer freeWEString(cfn)
	assertValueOwnersAlive(params)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	owner, release, finishDependencies, err := vm.beginAsyncRun(ctx, params, ast)
	if err != nil {
		return nil, fmt.Errorf("run AST module asynchronously: %w", err)
	}
	defer runtime.KeepAlive(vm)
	defer runtime.KeepAlive(ast)
	ptr := C.WasmEdge_VMAsyncRunWasmFromASTModule(
		vm.ptr, ast.ptr, cfn, valuesPtr(cparams), paramCount)
	if ptr == nil {
		finishDependencies(false)
		release()
		return nil, fmt.Errorf("start async AST run: %w", ErrUnavailable)
	}
	finishDependencies(true)
	return newExecution(ptr, owner, params, release, vm.gate.poison), nil
}

func (vm *VM) beginAsyncRun(
	ctx context.Context,
	params []Value,
	extra ...any,
) (*generationGuard, func(), func(bool), error) {
	releaseGate, err := vm.gate.begin()
	if err != nil {
		return nil, nil, nil, err
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			releaseGate()
			return nil, nil, nil, err
		}
	}
	finishDependencies, err := vm.prepareStoreDependencies()
	if err != nil {
		releaseGate()
		return nil, nil, nil, err
	}
	// A one-shot run may replace the active native module before this call
	// returns, so invalidate old views before starting the detached worker.
	vm.active.advance()
	owner := vm.activeGuard()
	finishReferences, err := owner.prepareInvocationReferences(params)
	if err != nil {
		finishDependencies(false)
		releaseGate()
		return nil, nil, nil, err
	}
	owners := append([]any{vm}, valueOwners(params)...)
	owners = append(owners, extra...)
	releaseLeases, err := acquireLeases(owners...)
	if err != nil {
		finishReferences(false)
		finishDependencies(false)
		releaseGate()
		return nil, nil, nil, err
	}
	releaseContext, err := vm.bindContext(ctx)
	if err != nil {
		releaseLeases()
		finishReferences(false)
		finishDependencies(false)
		releaseGate()
		return nil, nil, nil, err
	}
	release := func() {
		releaseContext()
		releaseLeases()
		for _, value := range extra {
			if free, ok := value.(func()); ok {
				free()
			}
		}
		releaseGate()
	}
	finishStart := func(started bool) {
		finishReferences(started)
		finishDependencies(started)
	}
	return owner, release, finishStart, nil
}

// RunBytesContext is RunBytes with context cancellation.
func (vm *VM) RunBytesContext(
	ctx context.Context, b []byte, fn string, params ...Value,
) ([]Value, error) {
	if ctx == nil {
		return nil, fmt.Errorf("run bytes context is nil: %w", ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	execution, err := vm.runBytesAsync(ctx, b, fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, execution)
}

// RunFileContext is RunFile with context cancellation.
func (vm *VM) RunFileContext(
	ctx context.Context, path, fn string, params ...Value,
) ([]Value, error) {
	if ctx == nil {
		return nil, fmt.Errorf("run file context is nil: %w", ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	execution, err := vm.runFileAsync(ctx, path, fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, execution)
}

// RunContext is Run with context cancellation.
func (vm *VM) RunContext(
	ctx context.Context, ast *ASTModule, fn string, params ...Value,
) ([]Value, error) {
	if ctx == nil {
		return nil, fmt.Errorf("run context is nil: %w", ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	execution, err := vm.runAsync(ctx, ast, fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, execution)
}

// ExecuteAsync starts an asynchronous invocation of an exported function
// of the active module.
func (vm *VM) ExecuteAsync(fn string, params ...Value) (*Execution, error) {
	//nolint:staticcheck // A nil context is the internal no-cancellation sentinel.
	return vm.executeAsync(nil, fn, params...)
}

func (vm *VM) executeAsync(
	ctx context.Context, fn string, params ...Value,
) (*Execution, error) {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	assertValueOwnersAlive(params)
	cname := newWEString(fn)
	defer freeWEString(cname)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	ft, ok := functionTypeFromC(C.WasmEdge_VMGetFunctionType(vm.ptr, cname))
	if !ok {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "wasm function not found: " + fn}
	}
	if err := validateInvocationParams(ft, params); err != nil {
		return nil, err
	}
	owner := vm.activeGuard()
	release, finishReferences, err := vm.beginAsync(ctx, owner, params)
	if err != nil {
		return nil, fmt.Errorf("execute function asynchronously: %w", err)
	}
	ptr := C.WasmEdge_VMAsyncExecute(vm.ptr, cname,
		valuesPtr(cparams), paramCount)
	if ptr == nil {
		finishReferences(false)
		release()
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "async execution failed to start"}
	}
	finishReferences(true)
	return newExecution(ptr, owner, params, release, vm.gate.poison), nil
}

// ExecuteRegisteredAsync starts an asynchronous invocation of fn from the
// registered module named module.
func (vm *VM) ExecuteRegisteredAsync(module, fn string, params ...Value) (*Execution, error) {
	//nolint:staticcheck // A nil context is the internal no-cancellation sentinel.
	return vm.executeRegisteredAsync(nil, module, fn, params...)
}

func (vm *VM) executeRegisteredAsync(
	ctx context.Context, module, fn string, params ...Value,
) (*Execution, error) {
	vm.assertAlive()
	assertValueOwnersAlive(params)
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	defer runtime.KeepAlive(vm)
	cmodule := newWEString(module)
	defer freeWEString(cmodule)
	cfn := newWEString(fn)
	defer freeWEString(cfn)
	modulePtr := C.WasmEdge_VMGetRegisteredModule(vm.ptr, cmodule)
	if modulePtr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "registered wasm module not found: " + module}
	}
	owner := vm.registeredOwner(modulePtr)
	ft, ok := functionTypeFromC(
		C.WasmEdge_VMGetFunctionTypeRegistered(vm.ptr, cmodule, cfn))
	if !ok {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound,
			Message: "registered wasm function not found: " + module + "." + fn}
	}
	if err := validateInvocationParams(ft, params); err != nil {
		return nil, err
	}
	release, finishReferences, err := vm.beginAsync(ctx, owner, params)
	if err != nil {
		return nil, fmt.Errorf("execute registered function asynchronously: %w", err)
	}
	ptr := C.WasmEdge_VMAsyncExecuteRegistered(
		vm.ptr, cmodule, cfn, valuesPtr(cparams), paramCount)
	if ptr == nil {
		finishReferences(false)
		release()
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "registered async execution failed to start"}
	}
	finishReferences(true)
	return newExecution(ptr, owner, params, release, vm.gate.poison), nil
}

func (vm *VM) beginAsync(
	ctx context.Context,
	owner any,
	params []Value,
) (func(), func(bool), error) {
	releaseGate, err := vm.gate.begin()
	if err != nil {
		return nil, nil, err
	}
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			releaseGate()
			return nil, nil, err
		}
	}
	retainer, ok := owner.(invocationReferenceRetainer)
	if !ok {
		releaseGate()
		return nil, nil, ErrOwnership
	}
	finishReferences, err := retainer.prepareInvocationReferences(params)
	if err != nil {
		releaseGate()
		return nil, nil, err
	}
	owners := append([]any{vm}, valueOwners(params)...)
	if !ownerChainContains(owner, vm) {
		owners = append(owners, owner)
	}
	releaseLeases, err := acquireLeases(owners...)
	if err != nil {
		finishReferences(false)
		releaseGate()
		return nil, nil, err
	}
	releaseContext, err := vm.bindContext(ctx)
	if err != nil {
		releaseLeases()
		finishReferences(false)
		releaseGate()
		return nil, nil, err
	}
	return func() {
		releaseContext()
		releaseLeases()
		releaseGate()
	}, finishReferences, nil
}

// ExecuteContext is Execute with cancellation: when ctx fires, the WASM
// execution is interrupted and ctx's error is returned.
func (vm *VM) ExecuteContext(ctx context.Context, fn string, params ...Value) ([]Value, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execute context is nil: %w", ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ex, err := vm.executeAsync(ctx, fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, ex)
}

// ExecuteRegisteredContext is ExecuteRegistered with context cancellation.
func (vm *VM) ExecuteRegisteredContext(
	ctx context.Context, module, fn string, params ...Value,
) ([]Value, error) {
	if ctx == nil {
		return nil, fmt.Errorf("execute registered context is nil: %w", ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ex, err := vm.executeRegisteredAsync(ctx, module, fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, ex)
}

// ---- introspection ---------------------------------------------------------

// VMFunction describes one exported function of the active module.
type VMFunction struct {
	Name string
	Type FunctionType
}

// Functions lists the exported functions of the active module.
func (vm *VM) Functions() []VMFunction {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	names, types := snapshotParallelLists(
		func() uint32 {
			return uint32(C.WasmEdge_VMGetFunctionListLength(vm.ptr))
		},
		func(
			names []C.WasmEdge_String,
			types []*C.WasmEdge_FunctionTypeContext,
		) uint32 {
			return uint32(C.WasmEdge_VMGetFunctionList(
				vm.ptr, &names[0], &types[0], C.uint32_t(len(names)),
			))
		},
	)
	out := make([]VMFunction, 0, len(names))
	for i := range names {
		typ, _ := functionTypeFromC(types[i])
		out = append(out, VMFunction{
			Name: goString(names[i]),
			Type: typ,
		})
	}
	return out
}

// FunctionType returns a copied descriptor of an exported function of the
// active module, with ok=false when the export does not exist.
func (vm *VM) FunctionType(fn string) (FunctionType, bool) {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	cname := newWEString(fn)
	defer freeWEString(cname)
	return functionTypeFromC(C.WasmEdge_VMGetFunctionType(vm.ptr, cname))
}

// RegisteredFunctionType returns a copied descriptor of fn in a registered
// module, with ok=false when either name does not exist.
func (vm *VM) RegisteredFunctionType(module, fn string) (FunctionType, bool) {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	cmodule := newWEString(module)
	defer freeWEString(cmodule)
	cfn := newWEString(fn)
	defer freeWEString(cfn)
	return functionTypeFromC(
		C.WasmEdge_VMGetFunctionTypeRegistered(vm.ptr, cmodule, cfn))
}

// ActiveModule returns the anonymous active module instance (borrowed), or
// nil before Instantiate. A later successful Instantiate, started one-shot
// Run*, Reset, or Close invalidates the view and anything derived from it.
func (vm *VM) ActiveModule() *Module {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	return borrowedModule(C.WasmEdge_VMGetActiveModule(vm.ptr), vm.activeGuard())
}

// RegisteredModule looks up a module registered into this VM (borrowed).
// Reset or Close invalidates VM-owned module views. A caller-owned module in
// an external Store instead bounds the returned view by that module, which
// remains valid until its caller closes it.
func (vm *VM) RegisteredModule(name string) (*Module, bool) {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.WasmEdge_VMGetRegisteredModule(vm.ptr, cname)
	m := borrowedModule(ptr, vm.registeredOwner(ptr))
	return m, m != nil
}

// RegisteredModuleNames lists modules registered into this VM.
func (vm *VM) RegisteredModuleNames() []string {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_VMListRegisteredModuleLength(vm.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_VMListRegisteredModule(vm.ptr, buf, n)
		})
}

// Store returns the VM's store (borrowed).
func (vm *VM) Store() *Store {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	return &Store{
		ptr: C.WasmEdge_VMGetStoreContext(vm.ptr), life: borrowed(vm),
		state: vm.storeState,
		moduleOwner: func() any {
			return vm.registeredGuard()
		},
	}
}

// Loader returns the VM-owned loader (borrowed).
func (vm *VM) Loader() *Loader {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	return borrowedLoader(C.WasmEdge_VMGetLoaderContext(vm.ptr), vm)
}

// Validator returns the VM-owned validator (borrowed).
func (vm *VM) Validator() *Validator {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	return borrowedValidator(C.WasmEdge_VMGetValidatorContext(vm.ptr), vm)
}

// Executor returns the VM-owned executor (borrowed).
func (vm *VM) Executor() *Executor {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	executor := borrowedExecutor(C.WasmEdge_VMGetExecutorContext(vm.ptr), vm)
	if executor != nil {
		executor.gate = &vm.gate
	}
	return executor
}

// Stats returns the VM's statistics collector (borrowed).
func (vm *VM) Stats() *Statistics {
	vm.assertAlive()
	defer runtime.KeepAlive(vm)
	return borrowedStatistics(C.WasmEdge_VMGetStatisticsContext(vm.ptr), vm)
}

// Reset clears the VM back to the freshly created state (the C API's
// VMCleanup): the active module, loaded module and every non-built-in module
// in its store are dropped, import leases are released, and active/registered
// borrowed views are invalidated. WithExternalStore does not change this
// upstream behavior, so Reset also clears registrations that predate the VM.
// It returns ErrInUse rather than resetting while a dependant holds a VM
// lease (notably an in-flight asynchronous execution).
func (vm *VM) Reset() error {
	vm.assertAlive()
	if err := vm.ensureUnused(); err != nil {
		return err
	}
	defer runtime.KeepAlive(vm)
	C.WasmEdge_VMCleanup(vm.ptr)
	vm.active.invalidate()
	vm.registered.invalidate()
	for i := len(vm.importReleases) - 1; i >= 0; i-- {
		vm.importReleases[i]()
	}
	vm.importReleases = nil
	vm.releaseStoreDependencies()
	// WasmEdge 0.17.1 VMCleanup removes all non-built-in registrations from
	// the VM's store, including registrations that predate a VM using an
	// external store. Release the matching Go-side retention graph as well.
	vm.storeState.releaseAll()
	vm.ownedStoreModules = nil
	vm.imports = nil
	return nil
}

// Close destroys the VM (and everything it owns: internal store, loader,
// validator, executor, statistics, registered non-import modules). No-op
// after the first call.
func (vm *VM) Close() error {
	ptr := vm.ptr
	err := vm.life.close(func() {
		C.WasmEdge_VMDelete(ptr)
		vm.active.invalidate()
		vm.registered.invalidate()
		for i := len(vm.importReleases) - 1; i >= 0; i-- {
			vm.importReleases[i]()
		}
		vm.importReleases = nil
		vm.releaseStoreDependencies()
		vm.storeState.forgetLast(vm.ownedStoreModules)
		if vm.store == nil {
			vm.storeState.releaseAll()
		}
		vm.ownedStoreModules = nil
	})
	if err == nil {
		if vm.store != nil {
			vm.releaseStore()
			vm.releaseStore = nil
		}
		vm.store = nil
		vm.storeState = nil
		vm.imports = nil
	}
	return err
}
