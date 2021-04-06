package ssvm

// #include <ssvm.h>
import "C"
import "unsafe"

type Interpreter struct {
	_inner *C.SSVM_InterpreterContext
}

func NewInterpreter() *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithConfig(conf *Configure) *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithStatistics(stat *Statistics) *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(nil, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewInterpreterWithConfigAndStatistics(conf *Configure, stat *Statistics) *Interpreter {
	self := &Interpreter{
		_inner: C.SSVM_InterpreterCreate(conf._inner, stat._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Interpreter) Instantiate(store *Store, ast *AST) error {
	return newError(C.SSVM_InterpreterInstantiate(self._inner, store._inner, ast._inner))
}

func (self *Interpreter) RegisterImport(store *Store, imp *ImportObject) error {
	return newError(C.SSVM_InterpreterRegisterImport(self._inner, store._inner, imp._inner))
}

func (self *Interpreter) RegisterModule(store *Store, ast *AST, modname string) error {
	modstr := toSSVMStringWrap(modname)
	return newError(C.SSVM_InterpreterRegisterModule(self._inner, store._inner, ast._inner, modstr))
}

func (self *Interpreter) Invoke(store *Store, funcname string, params ...interface{}) ([]interface{}, error) {
	funcstr := toSSVMStringWrap(funcname)
	funccxt := store.FindFunction(funcname)
	ftype := funccxt.GetFunctionType()
	cparams := toSSVMValueSlide(params...)
	creturns := make([]C.SSVM_Value, len(ftype._returns))
	var ptrparams *C.SSVM_Value = nil
	var ptrreturns *C.SSVM_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.SSVM_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.SSVM_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.SSVM_InterpreterInvoke(self._inner, store._inner, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(ftype._returns)))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return fromSSVMValueSlide(creturns, ftype._returns), nil
}

func (self *Interpreter) InvokeRegistered(store *Store, modname string, funcname string, params ...interface{}) ([]interface{}, error) {
	modstr := toSSVMStringWrap(modname)
	funcstr := toSSVMStringWrap(funcname)
	funccxt := store.FindFunctionRegistered(modname, funcname)
	ftype := funccxt.GetFunctionType()
	cparams := toSSVMValueSlide(params...)
	creturns := make([]C.SSVM_Value, len(ftype._returns))
	var ptrparams *C.SSVM_Value = nil
	var ptrreturns *C.SSVM_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.SSVM_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.SSVM_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.SSVM_InterpreterInvokeRegistered(self._inner, store._inner, modstr, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(ftype._returns)))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return fromSSVMValueSlide(creturns, ftype._returns), nil
}

func (self *Interpreter) Delete() {
	C.SSVM_InterpreterDelete(self._inner)
	self._inner = nil
}
