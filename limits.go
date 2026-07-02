package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

// Limits describes the size bounds of a memory or table, in pages
// (memories) or elements (tables). The zero value means "min 0, no max,
// 32-bit, unshared".
//
// Since WasmEdge 0.17 the C API models this as a heap-allocated
// WasmEdge_LimitContext (Memory64 and Threads made it grow flags); the Go
// API keeps it a plain struct and materializes the C object only inside
// type constructors, so there is nothing for the caller to Close.
type Limits struct {
	Min    uint64
	Max    uint64 // meaningful only when HasMax
	HasMax bool
	Shared bool // Threads proposal: shared memory (requires HasMax)
	Is64   bool // Memory64 proposal: 64-bit addressing
}

// build materializes a caller-owned C limit; every consumer copies it
// (verified against lib/api/wasmedge.cpp), so callers free it right after
// the consuming call via the returned function.
func (l Limits) build() (*C.WasmEdge_LimitContext, func()) {
	var ptr *C.WasmEdge_LimitContext
	if l.HasMax {
		ptr = C.WasmEdge_LimitCreateWithMax(C.uint64_t(l.Min), C.uint64_t(l.Max),
			C.bool(l.Is64), C.bool(l.Shared))
	} else {
		ptr = C.WasmEdge_LimitCreate(C.uint64_t(l.Min), C.bool(l.Is64))
	}
	return ptr, func() {
		if ptr != nil {
			C.WasmEdge_LimitDelete(ptr)
		}
	}
}

// limitsFromC copies a borrowed C limit view into the plain struct.
func limitsFromC(ptr *C.WasmEdge_LimitContext) Limits {
	if ptr == nil {
		return Limits{}
	}
	l := Limits{
		Min:    uint64(C.WasmEdge_LimitGetMin(ptr)),
		HasMax: bool(C.WasmEdge_LimitHasMax(ptr)),
		Shared: bool(C.WasmEdge_LimitIsShared(ptr)),
		Is64:   bool(C.WasmEdge_LimitIs64Bit(ptr)),
	}
	if l.HasMax {
		l.Max = uint64(C.WasmEdge_LimitGetMax(ptr))
	}
	return l
}
