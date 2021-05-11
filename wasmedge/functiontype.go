package wasmedge

// #include <wasmedge.h>
import "C"

type FunctionType struct {
	_params  []C.enum_WasmEdge_ValType
	_returns []C.enum_WasmEdge_ValType
}

func NewFunctionType(params []ValType, returns []ValType) *FunctionType {
	self := &FunctionType{
		_params:  make([]C.enum_WasmEdge_ValType, len(params)),
		_returns: make([]C.enum_WasmEdge_ValType, len(returns)),
	}
	for i, t := range params {
		self._params[i] = C.enum_WasmEdge_ValType(t)
	}
	for i, t := range returns {
		self._returns[i] = C.enum_WasmEdge_ValType(t)
	}
	return self
}

func fromWasmEdgeFunctionType(cftype *C.WasmEdge_FunctionTypeContext) *FunctionType {
	ftype := &FunctionType{}
	if cftype != nil {
		lparams := C.WasmEdge_FunctionTypeGetParametersLength(cftype)
		lreturns := C.WasmEdge_FunctionTypeGetReturnsLength(cftype)
		if uint(lparams) > 0 {
			ftype._params = make([]C.enum_WasmEdge_ValType, uint(lparams))
			C.WasmEdge_FunctionTypeGetParameters(cftype, &ftype._params[0], lparams)
		}
		if uint(lreturns) > 0 {
			ftype._returns = make([]C.enum_WasmEdge_ValType, uint(lreturns))
			C.WasmEdge_FunctionTypeGetReturns(cftype, &ftype._returns[0], lreturns)
		}
	}
	return ftype
}

func toWasmEdgeFunctionType(ftype *FunctionType) *C.WasmEdge_FunctionTypeContext {
	var ptrparams *C.enum_WasmEdge_ValType = nil
	var ptrreturns *C.enum_WasmEdge_ValType = nil
	if len(ftype._params) > 0 {
		ptrparams = &(ftype._params[0])
	}
	if len(ftype._returns) > 0 {
		ptrreturns = &(ftype._returns[0])
	}
	return C.WasmEdge_FunctionTypeCreate(ptrparams, C.uint32_t(len(ftype._params)), ptrreturns, C.uint32_t(len(ftype._returns)))
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
