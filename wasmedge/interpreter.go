package wasmedge

// #include <wasmedge.h>
import "C"
import "unsafe"

type Interpreter struct {
	_inner *C.WasmEdge_InterpreterContext
}

func NewInterpreter() *Interpreter {
	self := &Interpreter{
		_inner: C.WasmEdge_InterpreterCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithConfig(conf *Configure) *Interpreter {
	self := &Interpreter{
		_inner: C.WasmEdge_InterpreterCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithStatistics(stat *Statistics) *Interpreter {
	self := &Interpreter{
		_inner: C.WasmEdge_InterpreterCreate(nil, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithConfigAndStatistics(conf *Configure, stat *Statistics) *Interpreter {
	self := &Interpreter{
		_inner: C.WasmEdge_InterpreterCreate(conf._inner, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Interpreter) Instantiate(store *Store, ast *AST) error {
	return newError(C.WasmEdge_InterpreterInstantiate(self._inner, store._inner, ast._inner))
}

func (self *Interpreter) RegisterImport(store *Store, imp *ImportObject) error {
	return newError(C.WasmEdge_InterpreterRegisterImport(self._inner, store._inner, imp._inner))
}

func (self *Interpreter) RegisterModule(store *Store, ast *AST, modname string) error {
	modstr := toWasmEdgeStringWrap(modname)
	return newError(C.WasmEdge_InterpreterRegisterModule(self._inner, store._inner, ast._inner, modstr))
}

func (self *Interpreter) Invoke(store *Store, funcname string, params ...interface{}) ([]interface{}, error) {
	funcstr := toWasmEdgeStringWrap(funcname)
	funccxt := store.FindFunction(funcname)
	ftype := funccxt.GetFunctionType()
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, len(ftype._returns))
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_InterpreterInvoke(self._inner, store._inner, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(ftype._returns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns, ftype._returns), nil
}

func (self *Interpreter) InvokeRegistered(store *Store, modname string, funcname string, params ...interface{}) ([]interface{}, error) {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	funccxt := store.FindFunctionRegistered(modname, funcname)
	ftype := funccxt.GetFunctionType()
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, len(ftype._returns))
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_InterpreterInvokeRegistered(self._inner, store._inner, modstr, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(ftype._returns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns, ftype._returns), nil
}

func (self *Interpreter) Delete() {
	C.WasmEdge_InterpreterDelete(self._inner)
	self._inner = nil
}
