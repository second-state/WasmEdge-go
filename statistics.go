package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Statistics collects execution counters when attached to an Executor (via
// WithStats) and enabled in Config.Stats. A VM owns its own instance,
// reachable through VM.Stats.
type Statistics struct {
	ptr          *C.WasmEdge_StatisticsContext
	life         lifetime
	costTable    []uint64
	costLimit    uint64
	hasCostLimit bool
}

// NewStatistics creates an owned statistics collector. WasmEdge treats
// allocation failure as an unrecoverable condition for this otherwise
// infallible constructor.
func NewStatistics() *Statistics {
	ptr := C.WasmEdge_StatisticsCreate()
	if ptr == nil {
		panic("wasmedge: native Statistics allocation failed")
	}
	s := &Statistics{ptr: ptr}
	arm(s, &s.life, "Statistics", func() { C.WasmEdge_StatisticsDelete(ptr) })
	return s
}

func borrowedStatistics(ptr *C.WasmEdge_StatisticsContext, owner any) *Statistics {
	if ptr == nil {
		return nil
	}
	return &Statistics{ptr: ptr, life: borrowed(owner)}
}

func (s *Statistics) assertAlive() { s.life.assertAlive("Statistics") }

func (s *Statistics) acquireLease() (func(), error) {
	return s.life.acquire("Statistics")
}

// InstrCount returns the executed-instruction count (requires
// Config.Stats.InstructionCounting).
func (s *Statistics) InstrCount() uint64 {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	return uint64(C.WasmEdge_StatisticsGetInstrCount(s.ptr))
}

// InstrPerSecond returns the average executed instructions per second
// (requires instruction counting and time measuring).
func (s *Statistics) InstrPerSecond() float64 {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	return float64(C.WasmEdge_StatisticsGetInstrPerSecond(s.ptr))
}

// TotalCost returns the accumulated cost (requires Config.Stats.CostMeasuring).
func (s *Statistics) TotalCost() uint64 {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	return uint64(C.WasmEdge_StatisticsGetTotalCost(s.ptr))
}

// SetCostTable replaces the per-opcode instruction costs. WasmEdge fills
// entries beyond the supplied slice with zero; an empty slice therefore
// installs an all-zero table. Counts that the native uint32 ABI cannot
// represent match ErrInvalidArgument.
func (s *Statistics) SetCostTable(costs []uint64) error {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	if _, err := checkedUint32Count("statistics cost", uint64(len(costs))); err != nil {
		return err
	}
	s.costTable = append(s.costTable[:0], costs...)
	s.applyCostTable()
	return nil
}

func (s *Statistics) applyCostTable() {
	if len(s.costTable) == 0 {
		C.WasmEdge_StatisticsSetCostTable(s.ptr, nil, 0)
		return
	}
	buf := make([]C.uint64_t, len(s.costTable))
	for i, cost := range s.costTable {
		buf[i] = C.uint64_t(cost)
	}
	count, err := checkedUint32Count("statistics cost", uint64(len(buf)))
	if err != nil {
		panic("wasmedge: internally stored cost table exceeds the native uint32 limit")
	}
	C.WasmEdge_StatisticsSetCostTable(s.ptr, &buf[0], C.uint32_t(count))
	runtime.KeepAlive(buf)
}

// SetCostLimit sets the maximum cumulative instruction cost. Execution stops
// with ErrCodeCostLimitExceeded when the limit is exceeded.
func (s *Statistics) SetCostLimit(limit uint64) {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	s.costLimit = limit
	s.hasCostLimit = true
	C.WasmEdge_StatisticsSetCostLimit(s.ptr, C.uint64_t(limit))
}

// reapplyPolicy restores settings that WasmEdge_ExecutorCreate resets on an
// externally supplied statistics context.
func (s *Statistics) reapplyPolicy() {
	if s == nil {
		return
	}
	if s.costTable != nil {
		s.applyCostTable()
	}
	if s.hasCostLimit {
		C.WasmEdge_StatisticsSetCostLimit(s.ptr, C.uint64_t(s.costLimit))
	}
}

// Clear resets all collected counters and accumulated cost.
func (s *Statistics) Clear() {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	C.WasmEdge_StatisticsClear(s.ptr)
}

// Close frees the collector. No-op for VM-owned views and after the first
// call.
func (s *Statistics) Close() error {
	ptr := s.ptr
	return s.life.close(func() { C.WasmEdge_StatisticsDelete(ptr) })
}
