package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Executor instantiates validated modules into a Store and invokes
// functions.
type Executor struct {
	ptr  *C.WasmEdge_ExecutorContext
	life lifetime
}

// ExecutorOption configures NewExecutor.
type ExecutorOption func(*executorOptions)

type executorOptions struct {
	stats *Statistics
}

// WithStats attaches a statistics collector. The collector remains
// caller-owned and must outlive the executor.
func WithStats(s *Statistics) ExecutorOption {
	return func(o *executorOptions) { o.stats = s }
}

// NewExecutor creates an executor honoring cfg (nil for defaults).
func NewExecutor(cfg *Config, opts ...ExecutorOption) (*Executor, error) {
	var o executorOptions
	for _, opt := range opts {
		opt(&o)
	}
	ccfg, free := cfg.build()
	defer free()
	var cstats *C.WasmEdge_StatisticsContext
	if o.stats != nil {
		cstats = o.stats.ptr
	}
	ptr := C.WasmEdge_ExecutorCreate(ccfg, cstats)
	runtime.KeepAlive(o.stats)
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError, Message: "executor creation failed"}
	}
	e := &Executor{ptr: ptr}
	arm(e, &e.life, "Executor", func() { C.WasmEdge_ExecutorDelete(ptr) })
	return e, nil
}

func borrowedExecutor(ptr *C.WasmEdge_ExecutorContext) *Executor {
	if ptr == nil {
		return nil
	}
	return &Executor{ptr: ptr, life: borrowed()}
}

// Instantiate creates the anonymous active module instance of ast within
// store. The caller owns the result and must Close it after use; closing it
// before the store is fine (the store tracks named modules only).
func (e *Executor) Instantiate(store *Store, ast *ASTModule) (*Module, error) {
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(ast)
	var mod *C.WasmEdge_ModuleInstanceContext
	if err := newResult(C.WasmEdge_ExecutorInstantiate(e.ptr, &mod, store.ptr, ast.ptr)); err != nil {
		return nil, err
	}
	return ownedModule(mod), nil
}

// Register instantiates ast within store under name, making its exports
// importable by later instantiations. The caller owns the result and must
// keep it alive while the store uses it, then Close it.
func (e *Executor) Register(store *Store, ast *ASTModule, name string) (*Module, error) {
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(ast)
	cname := newWEString(name)
	defer freeWEString(cname)
	var mod *C.WasmEdge_ModuleInstanceContext
	if err := newResult(C.WasmEdge_ExecutorRegister(e.ptr, &mod, store.ptr, ast.ptr, cname)); err != nil {
		return nil, err
	}
	return ownedModule(mod), nil
}

// RegisterImport registers a host (or previously instantiated) module into
// store under the module's own name. The module remains caller-owned and
// must outlive the store's use of it.
func (e *Executor) RegisterImport(store *Store, mod *Module) error {
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(mod)
	return newResult(C.WasmEdge_ExecutorRegisterImport(e.ptr, store.ptr, mod.ptr))
}

// RegisterImportWithAlias registers mod under an alias instead of its own
// name (new in the 0.17 C API). The module remains caller-owned.
func (e *Executor) RegisterImportWithAlias(store *Store, mod *Module, alias string) error {
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(store)
	defer runtime.KeepAlive(mod)
	calias := newWEString(alias)
	defer freeWEString(calias)
	return newResult(C.WasmEdge_ExecutorRegisterImportWithAlias(e.ptr, store.ptr, mod.ptr, calias))
}

// Invoke calls a function instance synchronously. The number and types of
// params must match the function type; results are sized from it.
func (e *Executor) Invoke(fn *Function, params ...Value) ([]Value, error) {
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(fn)
	ft := fn.Type()
	nret := len(ft.Results())
	cparams := packValues(params)
	creturns := make([]C.WasmEdge_Value, max(nret, 1))
	err := newResult(C.WasmEdge_ExecutorInvoke(e.ptr, fn.ptr,
		valuesPtr(cparams), C.uint32_t(len(cparams)),
		&creturns[0], C.uint32_t(nret)))
	if err != nil {
		return nil, err
	}
	return unpackValues(creturns[:nret]), nil
}

// InvokeContext is added in async.go (Phase 5) on top of
// WasmEdge_ExecutorAsyncInvoke; see PLAN.md.

// Close frees the executor. Live host-function calls must not outlast it.
// No-op for borrowed views and after the first call.
func (e *Executor) Close() error {
	ptr := e.ptr
	return e.life.close(func() { C.WasmEdge_ExecutorDelete(ptr) })
}
