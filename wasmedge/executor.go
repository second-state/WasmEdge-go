package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import "unsafe"

type Executor struct {
	_inner *C.WasmEdge_ExecutorContext
}

func NewExecutor() *Executor {
	self := &Executor{
		_inner: C.WasmEdge_ExecutorCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewExecutorWithConfig(conf *Configure) *Executor {
	self := &Executor{
		_inner: C.WasmEdge_ExecutorCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewExecutorWithStatistics(stat *Statistics) *Executor {
	self := &Executor{
		_inner: C.WasmEdge_ExecutorCreate(nil, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewExecutorWithConfigAndStatistics(conf *Configure, stat *Statistics) *Executor {
	self := &Executor{
		_inner: C.WasmEdge_ExecutorCreate(conf._inner, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Executor) Instantiate(store *Store, ast *AST) error {
	res := C.WasmEdge_ExecutorInstantiate(self._inner, store._inner, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Executor) RegisterImport(store *Store, imp *ImportObject) error {
	res := C.WasmEdge_ExecutorRegisterImport(self._inner, store._inner, imp._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Executor) RegisterModule(store *Store, ast *AST, modname string) error {
	modstr := toWasmEdgeStringWrap(modname)
	res := C.WasmEdge_ExecutorRegisterModule(self._inner, store._inner, ast._inner, modstr)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Executor) Invoke(store *Store, funcname string, params ...interface{}) ([]interface{}, error) {
	funcstr := toWasmEdgeStringWrap(funcname)
	funccxt := store.FindFunction(funcname)
	ftype := funccxt.GetFunctionType()
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
		self._inner, store._inner, funcstr,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns), nil
}

func (self *Executor) InvokeRegistered(store *Store, modname string, funcname string, params ...interface{}) ([]interface{}, error) {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	funccxt := store.FindFunctionRegistered(modname, funcname)
	ftype := funccxt.GetFunctionType()
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
	res := C.WasmEdge_ExecutorInvokeRegistered(
		self._inner, store._inner, modstr, funcstr,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns), nil
}

func (self *Executor) Delete() {
	C.WasmEdge_ExecutorDelete(self._inner)
	self._inner = nil
}
