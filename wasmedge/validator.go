package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import "runtime"

type Validator struct {
	_inner *C.WasmEdge_ValidatorContext
	_own   bool
}

func NewValidator() *Validator {
	validator := C.WasmEdge_ValidatorCreate(nil)
	if validator == nil {
		return nil
	}
	res := &Validator{_inner: validator, _own: true}
	runtime.SetFinalizer(res, (*Validator).Release)
	return res
}

func NewValidatorWithConfig(conf *Configure) *Validator {
	validator := C.WasmEdge_ValidatorCreate(conf._inner)
	if validator == nil {
		return nil
	}
	res := &Validator{_inner: validator, _own: true}
	runtime.SetFinalizer(res, (*Validator).Release)
	return res
}

func (self *Validator) Validate(ast *AST) error {
	res := C.WasmEdge_ValidatorValidate(self._inner, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Validator) Release() {
	if self._own {
		C.WasmEdge_ValidatorDelete(self._inner)
	}
	runtime.SetFinalizer(self, nil)
	self._inner = nil
	self._own = false
}
