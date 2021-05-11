package wasmedge

// #include <wasmedge.h>
import "C"

type Global struct {
	_inner *C.WasmEdge_GlobalInstanceContext
}

func NewGlobal(val interface{}, vtype ValMut) *Global {
	cval := toWasmEdgeValue(val)
	self := &Global{
		_inner: C.WasmEdge_GlobalInstanceCreate(cval, C.enum_WasmEdge_Mutability(vtype)),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Global) GetValType() ValType {
	return ValType(C.WasmEdge_GlobalInstanceGetValType(self._inner))
}

func (self *Global) GetMutability() ValMut {
	return ValMut(C.WasmEdge_GlobalInstanceGetMutability(self._inner))
}

func (self *Global) GetValue() interface{} {
	cval := C.WasmEdge_GlobalInstanceGetValue(self._inner)
	return fromWasmEdgeValue(cval, cval.Type)
}

func (self *Global) SetValue(val interface{}) {
	C.WasmEdge_GlobalInstanceSetValue(self._inner, toWasmEdgeValue(val))
}

func (self *Global) Delete() {
	C.WasmEdge_GlobalInstanceDelete(self._inner)
	self._inner = nil
}
