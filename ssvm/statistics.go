package ssvm

// #include <ssvm.h>
import "C"

type Statistics struct {
	_inner *C.SSVM_StatisticsContext
}

func NewStatistics() *Statistics {
	self := &Statistics{
		_inner: C.SSVM_StatisticsCreate(),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Statistics) GetInstrCount() uint {
	return uint(C.SSVM_StatisticsGetInstrCount(self._inner))
}

func (self *Statistics) GetInstrPerSecond() float64 {
	return float64(C.SSVM_StatisticsGetInstrPerSecond(self._inner))
}

func (self *Statistics) GetTotalCost() uint {
	return uint(C.SSVM_StatisticsGetTotalCost(self._inner))
}

func (self *Statistics) SetCostTable(table []uint64) {
	var ptr *uint64 = nil
	if len(table) > 0 {
		ptr = &(table[0])
	}
	C.SSVM_StatisticsSetCostTable(self._inner, (*C.uint64_t)(ptr), C.uint32_t(len(table)))
}

func (self *Statistics) SetCostLimit(limit uint) {
	C.SSVM_StatisticsSetCostLimit(self._inner, C.uint64_t(limit))
}

func (self *Statistics) Delete() {
	C.SSVM_StatisticsDelete(self._inner)
	self._inner = nil
}
