package wasmedge

// #include <wasmedge.h>
import "C"
import "unsafe"

type Memory struct {
	_inner *C.WasmEdge_MemoryInstanceContext
}

func NewMemory(lim Limit) *Memory {
	climit := C.WasmEdge_Limit{HasMax: C.bool(lim.hasmax), Min: C.uint32_t(lim.min), Max: C.uint32_t(lim.max)}
	self := &Memory{
		_inner: C.WasmEdge_MemoryInstanceCreate(climit),
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
	res := C.WasmEdge_MemoryInstanceGetData(self._inner, ptrdata, C.uint32_t(off), C.uint32_t(length))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}

	return data, nil
}

func (self *Memory) SetData(data []byte, off uint, length uint) error {
	var ptrdata *C.uint8_t = nil
	if len(data) > 0 {
		ptrdata = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	res := C.WasmEdge_MemoryInstanceSetData(self._inner, ptrdata, C.uint32_t(off), C.uint32_t(length))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Memory) GetPageSize() uint {
	return uint(C.WasmEdge_MemoryInstanceGetPageSize(self._inner))
}

func (self *Memory) GrowPage(size uint) error {
	res := C.WasmEdge_MemoryInstanceGrowPage(self._inner, C.uint32_t(size))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Memory) Delete() {
	C.WasmEdge_MemoryInstanceDelete(self._inner)
	self._inner = nil
}
