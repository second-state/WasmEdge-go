package ssvm

// #include <ssvm.h>
import "C"
import "unsafe"

type Memory struct {
	_inner *C.SSVM_MemoryInstanceContext
}

func NewMemory(lim Limit) *Memory {
	climit := C.SSVM_Limit{HasMax: C.bool(lim.hasmax), Min: C.uint32_t(lim.min), Max: C.uint32_t(lim.max)}
	self := &Memory{
		_inner: C.SSVM_MemoryInstanceCreate(climit),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Memory) GetData(off uint, length uint) ([]byte, error) {
	data := make([]byte, length)
	var ptrdata *C.uint8_t = nil
	if len(data) > 0 {
		ptrdata = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	res := C.SSVM_MemoryInstanceGetData(self._inner, ptrdata, C.uint32_t(off), C.uint32_t(length))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}

	return data, nil
}

func (self *Memory) SetData(data []byte, off uint, length uint) error {
	var ptrdata *C.uint8_t = nil
	if len(data) > 0 {
		ptrdata = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	return newError(C.SSVM_MemoryInstanceSetData(self._inner, ptrdata, C.uint32_t(off), C.uint32_t(length)))
}

func (self *Memory) GetPageSize() uint {
	return uint(C.SSVM_MemoryInstanceGetPageSize(self._inner))
}

func (self *Memory) GrowPage(size uint) error {
	return newError(C.SSVM_MemoryInstanceGrowPage(self._inner, C.uint32_t(size)))
}

func (self *Memory) Delete() {
	C.SSVM_MemoryInstanceDelete(self._inner)
	self._inner = nil
}
