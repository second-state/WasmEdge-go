package ssvm

// #include <ssvm.h>
import "C"

type Table struct {
	_inner *C.SSVM_TableInstanceContext
}

func NewTable(rtype RefType, lim Limit) *Table {
	climit := C.SSVM_Limit{HasMax: C.bool(lim.hasmax), Min: C.uint32_t(lim.min), Max: C.uint32_t(lim.max)}
	crtype := C.enum_SSVM_RefType(rtype)
	self := &Table{
		_inner: C.SSVM_TableInstanceCreate(crtype, climit),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Table) GetRefType() RefType {
	return RefType(C.SSVM_TableInstanceGetRefType(self._inner))
}

func (self *Table) GetData(off uint) (interface{}, error) {
	cval := C.SSVM_Value{}
	res := C.SSVM_TableInstanceGetData(self._inner, &cval, C.uint32_t(off))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return fromSSVMValue(cval, cval.Type), nil
}

func (self *Table) SetData(data interface{}, off uint) error {
	cval := toSSVMValue(data)
	return newError(C.SSVM_TableInstanceSetData(self._inner, cval, C.uint32_t(off)))
}

func (self *Table) GetSize() uint {
	return uint(C.SSVM_TableInstanceGetSize(self._inner))
}

func (self *Table) Grow(size uint) error {
	return newError(C.SSVM_TableInstanceGrow(self._inner, C.uint32_t(size)))
}

func (self *Table) Delete() {
	C.SSVM_TableInstanceDelete(self._inner)
	self._inner = nil
}
