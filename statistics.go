package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Statistics collects execution counters when attached to an Executor (via
// WithStats) and enabled in Config.Stats. A VM owns its own instance,
// reachable through VM.Stats.
type Statistics struct {
	ptr  *C.WasmEdge_StatisticsContext
	life lifetime
}

// NewStatistics creates an owned statistics collector.
func NewStatistics() *Statistics {
	ptr := C.WasmEdge_StatisticsCreate()
	if ptr == nil {
		return nil
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

// InstrCount returns the executed-instruction count (requires
// Config.Stats.InstructionCounting).
func (s *Statistics) InstrCount() uint64 {
	defer runtime.KeepAlive(s)
	return uint64(C.WasmEdge_StatisticsGetInstrCount(s.ptr))
}

// InstrPerSecond returns the average executed instructions per second
// (requires instruction counting and time measuring).
func (s *Statistics) InstrPerSecond() float64 {
	defer runtime.KeepAlive(s)
	return float64(C.WasmEdge_StatisticsGetInstrPerSecond(s.ptr))
}

// TotalCost returns the accumulated cost (requires Config.Stats.CostMeasuring).
func (s *Statistics) TotalCost() uint64 {
	defer runtime.KeepAlive(s)
	return uint64(C.WasmEdge_StatisticsGetTotalCost(s.ptr))
}

// TODO(intern-easy): A2 — bind the remaining statistics APIs on this type:
//
//	SetCostTable(costs []uint64)  -> WasmEdge_StatisticsSetCostTable
//	SetCostLimit(limit uint64)    -> WasmEdge_StatisticsSetCostLimit
//	Clear()                       -> WasmEdge_StatisticsClear
//
// Pattern: InstrCount above (KeepAlive discipline!); for the slice argument
// mimic packValues in value.go (&buf[0] + length). Extend
// TestStatistics in pipeline_test.go: set a cost limit of 1 with cost
// measuring enabled, run the Fib fixture, and assert
// errors.Is(err, resultError-style *Error with ErrCodeCostLimitExceeded).

// Close frees the collector. No-op for VM-owned views and after the first
// call.
func (s *Statistics) Close() error {
	ptr := s.ptr
	return s.life.close(func() { C.WasmEdge_StatisticsDelete(ptr) })
}
