package wasmedge

// #include <wasmedge.h>
import "C"

type Table struct {
	_inner *C.WasmEdge_TableInstanceContext
}

func NewTable(rtype RefType, lim Limit) *Table {
	climit := C.WasmEdge_Limit{HasMax: C.bool(lim.hasmax), Min: C.uint32_t(lim.min), Max: C.uint32_t(lim.max)}
	crtype := C.enum_WasmEdge_RefType(rtype)
	self := &Table{
		_inner: C.WasmEdge_TableInstanceCreate(crtype, climit),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Table) GetRefType() RefType {
	return RefType(C.WasmEdge_TableInstanceGetRefType(self._inner))
}

func (self *Table) GetData(off uint) (interface{}, error) {
	cval := C.WasmEdge_Value{}
	res := C.WasmEdge_TableInstanceGetData(self._inner, &cval, C.uint32_t(off))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValue(cval, cval.Type), nil
}

func (self *Table) SetData(data interface{}, off uint) error {
	cval := toWasmEdgeValue(data)
	return newError(C.WasmEdge_TableInstanceSetData(self._inner, cval, C.uint32_t(off)))
}

func (self *Table) GetSize() uint {
	return uint(C.WasmEdge_TableInstanceGetSize(self._inner))
}

func (self *Table) Grow(size uint) error {
	return newError(C.WasmEdge_TableInstanceGrow(self._inner, C.uint32_t(size)))
}

func (self *Table) Delete() {
	C.WasmEdge_TableInstanceDelete(self._inner)
	self._inner = nil
}
