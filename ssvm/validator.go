package ssvm

// #include <ssvm.h>
import "C"

type Validator struct {
	_inner *C.SSVM_ValidatorContext
}

func NewValidator() *Validator {
	self := &Validator{
		_inner: C.SSVM_ValidatorCreate(nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewValidatorWithConfig(conf *Configure) *Validator {
	self := &Validator{
		_inner: C.SSVM_ValidatorCreate(conf._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Validator) Validate(ast *AST) error {
	return newError(C.SSVM_ValidatorValidate(self._inner, ast._inner))
}

func (self *Validator) Delete() {
	C.SSVM_ValidatorDelete(self._inner)
	self._inner = nil
}
