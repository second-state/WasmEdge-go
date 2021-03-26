package ssvm

// #include <ssvm.h>
import "C"

type FunctionType struct {
	_params  []C.enum_SSVM_ValType
	_returns []C.enum_SSVM_ValType
}

func NewFunctionType(params []ValType, returns []ValType) *FunctionType {
	self := &FunctionType{
		_params:  make([]C.enum_SSVM_ValType, len(params)),
		_returns: make([]C.enum_SSVM_ValType, len(returns)),
	}
	for i, t := range params {
		self._params[i] = C.enum_SSVM_ValType(t)
	}
	for i, t := range returns {
		self._returns[i] = C.enum_SSVM_ValType(t)
	}
	return self
}

func fromSSVMFunctionType(cftype *C.SSVM_FunctionTypeContext) *FunctionType {
	ftype := &FunctionType{}
	if cftype != nil {
		lparams := C.SSVM_FunctionTypeGetParametersLength(cftype)
		lreturns := C.SSVM_FunctionTypeGetReturnsLength(cftype)
		if uint(lparams) > 0 {
			ftype._params = make([]C.enum_SSVM_ValType, uint(lparams))
			C.SSVM_FunctionTypeGetParameters(cftype, &ftype._params[0], lparams)
		}
		if uint(lreturns) > 0 {
			ftype._returns = make([]C.enum_SSVM_ValType, uint(lreturns))
			C.SSVM_FunctionTypeGetReturns(cftype, &ftype._returns[0], lreturns)
		}
	}
	return ftype
}

func toSSVMFunctionType(ftype *FunctionType) *C.SSVM_FunctionTypeContext {
	var ptrparams *C.enum_SSVM_ValType = nil
	var ptrreturns *C.enum_SSVM_ValType = nil
	if len(ftype._params) > 0 {
		ptrparams = &(ftype._params[0])
	}
	if len(ftype._returns) > 0 {
		ptrreturns = &(ftype._returns[0])
	}
	return C.SSVM_FunctionTypeCreate(ptrparams, C.uint32_t(len(ftype._params)), ptrreturns, C.uint32_t(len(ftype._returns)))
}

func (self *FunctionType) GetParameters() []ValType {
	valtype := make([]ValType, len(self._params))
	for i, val := range self._params {
		valtype[i] = ValType(val)
	}
	return valtype
}

func (self *FunctionType) GetReturns() []ValType {
	valtype := make([]ValType, len(self._returns))
	for i, val := range self._returns {
		valtype[i] = ValType(val)
	}
	return valtype
}
