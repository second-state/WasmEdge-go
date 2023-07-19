package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"unsafe"
)

type Executor struct {
	_inner *C.WasmEdge_ExecutorContext
	_own   bool
}

func NewExecutor() *Executor {
	executor := C.WasmEdge_ExecutorCreate(nil, nil)
	if executor == nil {
		return nil
	}
	return &Executor{_inner: executor, _own: true}
}

func NewExecutorWithConfig(conf *Configure) *Executor {
	executor := C.WasmEdge_ExecutorCreate(conf._inner, nil)
	if executor == nil {
		return nil
	}
	return &Executor{_inner: executor, _own: true}
}

func NewExecutorWithStatistics(stat *Statistics) *Executor {
	executor := C.WasmEdge_ExecutorCreate(nil, stat._inner)
	if executor == nil {
		return nil
	}
	return &Executor{_inner: executor, _own: true}
}

func NewExecutorWithConfigAndStatistics(conf *Configure, stat *Statistics) *Executor {
	executor := C.WasmEdge_ExecutorCreate(conf._inner, stat._inner)
	if executor == nil {
		return nil
	}
	return &Executor{_inner: executor, _own: true}
}

func (self *Executor) Instantiate(store *Store, ast *AST) (*Module, error) {
	var module *C.WasmEdge_ModuleInstanceContext = nil
	res := C.WasmEdge_ExecutorInstantiate(self._inner, &module, store._inner, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return &Module{_inner: module, _own: true}, nil
}

func (self *Executor) Register(store *Store, ast *AST, modname string) (*Module, error) {
	var module *C.WasmEdge_ModuleInstanceContext = nil
	modstr := toWasmEdgeStringWrap(modname)
	res := C.WasmEdge_ExecutorRegister(self._inner, &module, store._inner, ast._inner, modstr)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return &Module{_inner: module, _own: true}, nil
}

func (self *Executor) RegisterImport(store *Store, module *Module) error {
	res := C.WasmEdge_ExecutorRegisterImport(self._inner, store._inner, module._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Executor) Invoke(funcinst *Function, params ...interface{}) ([]interface{}, error) {
	ftype := funcinst.GetFunctionType()
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, ftype.GetReturnsLength())
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_ExecutorInvoke(
		self._inner, funcinst._inner,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns), nil
}

func (self *Executor) AsyncInvoke(funcinst *Function, params ...interface{}) *Async {
	cparams := toWasmEdgeValueSlide(params...)
	var ptrparams *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	async := C.WasmEdge_ExecutorAsyncInvoke(self._inner, funcinst._inner, ptrparams, C.uint32_t(len(cparams)))
	if async == nil {
		return nil
	}
	return &Async{_inner: async, _own: true}
}

func (self *Executor) Release() {
	if self._own {
		C.WasmEdge_ExecutorDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
