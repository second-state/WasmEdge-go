package ssvm

// #include <ssvm.h>
import "C"

type Function struct {
	_inner *C.SSVM_FunctionInstanceContext
}

func (self *Function) GetFunctionType() *FunctionType {
	return fromSSVMFunctionType(C.SSVM_FunctionInstanceGetFunctionType(self._inner))
}
