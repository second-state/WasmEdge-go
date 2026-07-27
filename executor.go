package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime"
)

// Executor instantiates validated modules into a Store and invokes
// functions.
type Executor struct {
	ptr          *C.WasmEdge_ExecutorContext
	life         lifetime
	gate         *executionGate
	stats        *Statistics
	releaseStats func()
}

// ExecutorOption configures NewExecutor.
type ExecutorOption func(*executorOptions)

type executorOptions struct {
	stats *Statistics
}

// WithStats attaches a statistics collector. The collector remains
// caller-owned; the executor leases it until Close, so an early collector
// Close returns ErrInUse.
func WithStats(s *Statistics) ExecutorOption {
	return func(o *executorOptions) { o.stats = s }
}

// NewExecutor creates an executor honoring cfg (nil for defaults).
func NewExecutor(cfg *Config, opts ...ExecutorOption) (*Executor, error) {
	var o executorOptions
	for i, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("executor option %d is nil: %w", i, ErrInvalidArgument)
		}
		opt(&o)
	}
	ccfg, free, err := cfg.build()
	if err != nil {
		return nil, fmt.Errorf("create executor configuration: %w", err)
	}
	defer free()

	var cstats *C.WasmEdge_StatisticsContext
	releaseStats := func() {}
	if o.stats != nil {
		releaseStats, err = o.stats.acquireLease()
		if err != nil {
			return nil, fmt.Errorf("attach statistics: %w", err)
		}
		cstats = o.stats.ptr
	}
	ptr := C.WasmEdge_ExecutorCreate(ccfg, cstats)
	runtime.KeepAlive(o.stats)
	if ptr == nil {
		releaseStats()
		return nil, fmt.Errorf("create executor: %w", ErrUnavailable)
	}
	if o.stats != nil {
		o.stats.reapplyPolicy()
	}
	e := &Executor{
		ptr: ptr, gate: &executionGate{},
		stats: o.stats, releaseStats: releaseStats,
	}
	arm(e, &e.life, "Executor", func() { C.WasmEdge_ExecutorDelete(ptr) })
	return e, nil
}

func borrowedExecutor(ptr *C.WasmEdge_ExecutorContext, owner any) *Executor {
	if ptr == nil {
		return nil
	}
	return &Executor{ptr: ptr, life: borrowed(owner), gate: &executionGate{}}
}

func (e *Executor) assertAlive() { e.life.assertAlive("Executor") }

func (e *Executor) acquireLease() (func(), error) {
	return e.life.acquire("Executor")
}

// Instantiate creates the anonymous active module instance of ast within
// store. The caller owns the result and must Close it after use; closing it
// before the store is fine (the store tracks named modules only).
func (e *Executor) Instantiate(store *Store, ast *ASTModule) (*Module, error) {
	e.assertAlive()
	if store == nil || ast == nil {
		return nil, fmt.Errorf("instantiate requires non-nil store and AST module: %w", ErrInvalidArgument)
	}
	store.assertAlive()
	ast.assertAlive()
	dependencyModules, releaseDependencies, err := store.acquireDependencyLeases()
	if err != nil {
		return nil, fmt.Errorf("lease imported modules: %w", err)
	}
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(ast)
	var mod *C.WasmEdge_ModuleInstanceContext
	if err := newResult(C.WasmEdge_ExecutorInstantiate(e.ptr, &mod, store.ptr, ast.ptr)); err != nil {
		releaseDependencies()
		return nil, err
	}
	owned := ownedModule(mod)
	owned.dependencies = append(owned.dependencies, releaseDependencies)
	owned.dependencyModules = append(owned.dependencyModules, dependencyModules...)
	return owned, nil
}

// Register instantiates ast within store under name, making its exports
// importable by later instantiations. The caller owns the result. Closing it
// automatically unregisters it from the store unless an instantiated
// dependant still holds a lease.
func (e *Executor) Register(store *Store, ast *ASTModule, name string) (*Module, error) {
	e.assertAlive()
	if store == nil || ast == nil {
		return nil, fmt.Errorf("register requires non-nil store and AST module: %w", ErrInvalidArgument)
	}
	store.assertAlive()
	ast.assertAlive()
	dependencyModules, releaseDependencies, err := store.acquireDependencyLeases()
	if err != nil {
		return nil, fmt.Errorf("lease imported modules: %w", err)
	}
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(ast)
	cname := newWEString(name)
	defer freeWEString(cname)
	var mod *C.WasmEdge_ModuleInstanceContext
	if err := newResult(C.WasmEdge_ExecutorRegister(e.ptr, &mod, store.ptr, ast.ptr, cname)); err != nil {
		releaseDependencies()
		return nil, err
	}
	owned := ownedModule(mod)
	owned.dependencies = append(owned.dependencies, releaseDependencies)
	owned.dependencyModules = append(owned.dependencyModules, dependencyModules...)
	store.retain(owned)
	return owned, nil
}

// RegisterImport registers a host (or previously instantiated) module into
// store under the module's own name. The module remains caller-owned.
// Module.Close automatically unregisters it, but returns ErrInUse while an
// instantiated dependant still imports it. Borrowed module views are rejected
// because their original owner could otherwise invalidate a Store alias.
func (e *Executor) RegisterImport(store *Store, mod *Module) error {
	e.assertAlive()
	if store == nil || mod == nil {
		return fmt.Errorf("register import requires non-nil store and module: %w", ErrInvalidArgument)
	}
	store.assertAlive()
	mod.assertAlive()
	if !mod.life.isOwned() {
		return fmt.Errorf("register import requires a caller-owned module: %w", ErrOwnership)
	}
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(mod)
	err := newResult(C.WasmEdge_ExecutorRegisterImport(e.ptr, store.ptr, mod.ptr))
	if err == nil {
		store.retain(mod)
	}
	return err
}

// RegisterImportWithAlias registers mod under an alias instead of its own
// name (new in the 0.17 C API). The module remains caller-owned and Close
// automatically unregisters all of its aliases unless a dependant leases it.
// Borrowed module views are rejected.
func (e *Executor) RegisterImportWithAlias(store *Store, mod *Module, alias string) error {
	e.assertAlive()
	if store == nil || mod == nil {
		return fmt.Errorf("register import alias requires non-nil store and module: %w", ErrInvalidArgument)
	}
	store.assertAlive()
	mod.assertAlive()
	if !mod.life.isOwned() {
		return fmt.Errorf("register import alias requires a caller-owned module: %w", ErrOwnership)
	}
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(mod)
	calias := newWEString(alias)
	defer freeWEString(calias)
	err := newResult(C.WasmEdge_ExecutorRegisterImportWithAlias(e.ptr, store.ptr, mod.ptr, calias))
	if err == nil {
		store.retain(mod)
	}
	return err
}

// Invoke calls a function instance synchronously. The number and types of
// params must match the function type; results are sized from it.
func (e *Executor) Invoke(fn *Function, params ...Value) ([]Value, error) {
	e.assertAlive()
	if fn == nil {
		return nil, fmt.Errorf("invoke requires a non-nil function: %w", ErrInvalidArgument)
	}
	fn.assertAlive()
	assertValueOwnersAlive(params)
	ft := fn.Type()
	if err := validateInvocationParams(ft, params); err != nil {
		return nil, err
	}
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	releaseGate, err := e.gate.begin()
	if err != nil {
		return nil, fmt.Errorf("invoke function: %w", err)
	}
	defer releaseGate()
	finishReferences, err := fn.prepareInvocationReferences(params)
	if err != nil {
		return nil, fmt.Errorf("retain invocation references: %w", err)
	}
	defer finishReferences(false)
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(fn)
	defer runtime.KeepAlive(params)
	nret := len(ft.Results)
	creturns := make([]C.WasmEdge_Value, max(nret, 1))
	err = newResult(C.WasmEdge_ExecutorInvoke(e.ptr, fn.ptr,
		valuesPtr(cparams), paramCount,
		&creturns[0], C.uint32_t(nret)))
	// Once native invocation begins, guest code may have persisted a
	// reference even when the eventual result is a trap.
	finishReferences(true)
	if err != nil {
		return nil, err
	}
	return unpackValuesOwned(creturns[:nret], fn, params...), nil
}

// InvokeContext is implemented in async.go over
// WasmEdge_ExecutorAsyncInvoke.

// Close frees the executor. Live host-function calls must not outlast it.
// No-op for borrowed views and after the first call.
func (e *Executor) Close() error {
	ptr := e.ptr
	err := e.life.close(func() { C.WasmEdge_ExecutorDelete(ptr) })
	if err == nil && e.stats != nil {
		e.releaseStats()
		e.releaseStats = nil
		e.stats = nil
	}
	return err
}
