package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Validator struct {
	_inner *C.WasmEdge_ValidatorContext
	_own   bool
}

func NewValidator() *Validator {
	validator := C.WasmEdge_ValidatorCreate(nil)
	if validator == nil {
		return nil
	}
	return &Validator{_inner: validator, _own: true}
}

func NewValidatorWithConfig(conf *Configure) *Validator {
	validator := C.WasmEdge_ValidatorCreate(conf._inner)
	if validator == nil {
		return nil
	}
	return &Validator{_inner: validator, _own: true}
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
	self._inner = nil
	self._own = false
}
