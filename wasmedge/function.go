package wasmedge

// #include <wasmedge.h>
import "C"

type Function struct {
	_inner *C.WasmEdge_FunctionInstanceContext
}

func (self *Function) GetFunctionType() *FunctionType {
	return fromWasmEdgeFunctionType(C.WasmEdge_FunctionInstanceGetFunctionType(self._inner))
}
