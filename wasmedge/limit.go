package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Limit struct {
	_inner *C.WasmEdge_LimitContext
	_own   bool
}

func NewLimit(minVal uint) *Limit {
	limit := C.WasmEdge_LimitCreate(C.uint64_t(minVal), C.bool(false))
	if limit == nil {
		return nil
	}
	return &Limit{_inner: limit, _own: true}
}

func NewLimitWithMax(minVal uint, maxVal uint) *Limit {
	if maxVal < minVal {
		return nil
	}
	limit := C.WasmEdge_LimitCreateWithMax(
		C.uint64_t(minVal), C.uint64_t(maxVal), C.bool(false), C.bool(false))
	if limit == nil {
		return nil
	}
	return &Limit{_inner: limit, _own: true}
}

func NewLimitSharedWithMax(minVal uint, maxVal uint) *Limit {
	if maxVal < minVal {
		return nil
	}
	limit := C.WasmEdge_LimitCreateWithMax(
		C.uint64_t(minVal), C.uint64_t(maxVal), C.bool(false), C.bool(true))
	if limit == nil {
		return nil
	}
	return &Limit{_inner: limit, _own: true}
}

func NewLimit64(minVal uint) *Limit {
	limit := C.WasmEdge_LimitCreate(C.uint64_t(minVal), C.bool(true))
	if limit == nil {
		return nil
	}
	return &Limit{_inner: limit, _own: true}
}

func NewLimit64WithMax(minVal uint, maxVal uint) *Limit {
	if maxVal < minVal {
		return nil
	}
	limit := C.WasmEdge_LimitCreateWithMax(
		C.uint64_t(minVal), C.uint64_t(maxVal), C.bool(true), C.bool(false))
	if limit == nil {
		return nil
	}
	return &Limit{_inner: limit, _own: true}
}

func NewLimit64SharedWithMax(minVal uint, maxVal uint) *Limit {
	if maxVal < minVal {
		return nil
	}
	limit := C.WasmEdge_LimitCreateWithMax(
		C.uint64_t(minVal), C.uint64_t(maxVal), C.bool(true), C.bool(true))
	if limit == nil {
		return nil
	}
	return &Limit{_inner: limit, _own: true}
}

func (self *Limit) HasMax() bool {
	return bool(C.WasmEdge_LimitHasMax(self._inner))
}

func (self *Limit) IsShared() bool {
	return bool(C.WasmEdge_LimitIsShared(self._inner))
}

func (self *Limit) Is64Bit() bool {
	return bool(C.WasmEdge_LimitIs64Bit(self._inner))
}

func (self *Limit) GetMin() uint {
	return uint(C.WasmEdge_LimitGetMin(self._inner))
}

func (self *Limit) GetMax() uint {
	return uint(C.WasmEdge_LimitGetMax(self._inner))
}

func (self *Limit) IsEqual(lim *Limit) bool {
	return bool(C.WasmEdge_LimitIsEqual(self._inner, lim._inner))
}

func (self *Limit) Release() {
	if self._own {
		C.WasmEdge_LimitDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
