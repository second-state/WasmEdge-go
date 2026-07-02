package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Global is a global-variable instance.
type Global struct {
	ptr  *C.WasmEdge_GlobalInstanceContext
	life lifetime
}

// NewGlobal creates a standalone global for exporting from a host module.
// init must match the global type's value type; the GlobalType remains
// caller-owned.
func NewGlobal(gt *GlobalType, init Value) *Global {
	if gt == nil {
		return nil
	}
	ptr := C.WasmEdge_GlobalInstanceCreate(gt.ptr, init.raw)
	runtime.KeepAlive(gt)
	if ptr == nil {
		return nil
	}
	g := &Global{ptr: ptr}
	arm(g, &g.life, "Global", func() { C.WasmEdge_GlobalInstanceDelete(ptr) })
	return g
}

func borrowedGlobal(ptr *C.WasmEdge_GlobalInstanceContext) *Global {
	if ptr == nil {
		return nil
	}
	return &Global{ptr: ptr, life: borrowed()}
}

// Type returns the global's type (borrowed).
func (g *Global) Type() *GlobalType {
	defer runtime.KeepAlive(g)
	return borrowedGlobalType(C.WasmEdge_GlobalInstanceGetGlobalType(g.ptr))
}

// TODO(intern-medium): A7 — bind value access on this type:
//
//	Value() Value           -> WasmEdge_GlobalInstanceGetValue
//	SetValue(v Value) error -> WasmEdge_GlobalInstanceSetValue
//
// Mind the failure modes: setting a const global returns
// ErrCodeSetValueToConst and a type mismatch returns
// ErrCodeSetValueErrorType — assert both in the new TestGlobal in
// module_test.go, plus a v128 round-trip (pattern: TestValueRoundTrips in
// value_test.go). Pattern for the methods: Table.Get/Set in table.go.

// Close destroys an owned global that was not added to a module. No-op for
// borrowed views, after a transfer, and after the first call.
func (g *Global) Close() error {
	ptr := g.ptr
	return g.life.close(func() { C.WasmEdge_GlobalInstanceDelete(ptr) })
}
