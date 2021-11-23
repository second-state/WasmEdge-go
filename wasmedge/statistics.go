package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Statistics struct {
	_inner *C.WasmEdge_StatisticsContext
	_own   bool
}

func NewStatistics() *Statistics {
	stat := C.WasmEdge_StatisticsCreate()
	if stat == nil {
		return nil
	}
	return &Statistics{_inner: stat, _own: true}
}

func (self *Statistics) GetInstrCount() uint {
	return uint(C.WasmEdge_StatisticsGetInstrCount(self._inner))
}

func (self *Statistics) GetInstrPerSecond() float64 {
	return float64(C.WasmEdge_StatisticsGetInstrPerSecond(self._inner))
}

func (self *Statistics) GetTotalCost() uint {
	return uint(C.WasmEdge_StatisticsGetTotalCost(self._inner))
}

func (self *Statistics) SetCostTable(table []uint64) {
	var ptr *uint64 = nil
	if len(table) > 0 {
		ptr = &(table[0])
	}
	C.WasmEdge_StatisticsSetCostTable(self._inner, (*C.uint64_t)(ptr), C.uint32_t(len(table)))
}

func (self *Statistics) SetCostLimit(limit uint) {
	C.WasmEdge_StatisticsSetCostLimit(self._inner, C.uint64_t(limit))
}

func (self *Statistics) Release() {
	if self._own {
		C.WasmEdge_StatisticsDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
