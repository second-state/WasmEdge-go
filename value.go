package wasmedge

/*
#include <stdint.h>
#include <wasmedge/wasmedge.h>

// Handle<->pointer adapters for externref payloads. A cgo.Handle is an
// opaque uintptr, not a real pointer; going through C keeps go vet's
// unsafe.Pointer conversion checks quiet and documents the intent.
static void *wasmedgego_handleToPtr(uintptr_t H) { return (void *)H; }
static uintptr_t wasmedgego_ptrToHandle(void *P) { return (uintptr_t)P; }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Value is one WASM value together with its type. Values are small and
// copied freely; they manage no C memory.
//
// Accessors panic when the value has a different kind, mirroring the
// reflect package's contract: a kind mismatch is a programmer error, not a
// runtime condition to handle.
type Value struct {
	raw C.WasmEdge_Value
}

// I32 returns an i32 WASM value.
func I32(v int32) Value { return Value{raw: C.WasmEdge_ValueGenI32(C.int32_t(v))} }

// I64 returns an i64 WASM value.
func I64(v int64) Value { return Value{raw: C.WasmEdge_ValueGenI64(C.int64_t(v))} }

// F32 returns an f32 WASM value.
func F32(v float32) Value { return Value{raw: C.WasmEdge_ValueGenF32(C.float(v))} }

// F64 returns an f64 WASM value.
func F64(v float64) Value { return Value{raw: C.WasmEdge_ValueGenF64(C.double(v))} }

// V128 returns a v128 WASM value from its 16 lanes in little-endian lane
// order (lane 0 first), matching the memory layout used by all supported
// targets. cgo cannot pass __int128 through function calls, so the lanes
// are written into the value struct directly.
func V128(lanes [16]byte) Value {
	var v Value
	v.raw.Type = C.WasmEdge_ValTypeGenV128()
	*(*[16]byte)(unsafe.Pointer(&v.raw.Value)) = lanes
	return v
}

// ExternRefValue returns an externref WASM value carrying r. The *ExternRef
// must stay open while the engine can still observe the value.
func ExternRefValue(r *ExternRef) Value {
	if r == nil {
		return NullExternRef()
	}
	return Value{raw: C.WasmEdge_ValueGenExternRef(C.wasmedgego_handleToPtr(C.uintptr_t(r.handle())))}
}

// NullExternRef returns a null externref value.
func NullExternRef() Value {
	return Value{raw: C.WasmEdge_ValueGenExternRef(nil)}
}

// NullFuncRef returns a null funcref value.
func NullFuncRef() Value {
	return Value{raw: C.WasmEdge_ValueGenFuncRef(nil)}
}

// Type returns the value's WASM type.
func (v Value) Type() ValType { return ValType{raw: v.raw.Type} }

// Kind classifies the value; shorthand for v.Type().Kind().
func (v Value) Kind() ValKind { return v.Type().Kind() }

func (v Value) mustKind(k ValKind) {
	if got := v.Kind(); got != k {
		panic(fmt.Sprintf("wasmedge: Value kind is %s, not %s", got, k))
	}
}

// I32 returns the i32 payload. It panics if the value is not an i32.
func (v Value) I32() int32 {
	v.mustKind(ValKindI32)
	return int32(C.WasmEdge_ValueGetI32(v.raw))
}

// I64 returns the i64 payload. It panics if the value is not an i64.
func (v Value) I64() int64 {
	v.mustKind(ValKindI64)
	return int64(C.WasmEdge_ValueGetI64(v.raw))
}

// F32 returns the f32 payload. It panics if the value is not an f32.
func (v Value) F32() float32 {
	v.mustKind(ValKindF32)
	return float32(C.WasmEdge_ValueGetF32(v.raw))
}

// F64 returns the f64 payload. It panics if the value is not an f64.
func (v Value) F64() float64 {
	v.mustKind(ValKindF64)
	return float64(C.WasmEdge_ValueGetF64(v.raw))
}

// V128 returns the v128 lanes in little-endian lane order. It panics if the
// value is not a v128.
func (v Value) V128() [16]byte {
	v.mustKind(ValKindV128)
	return *(*[16]byte)(unsafe.Pointer(&v.raw.Value))
}

// IsNullRef reports whether the value is a null reference. Only meaningful
// for reference kinds.
func (v Value) IsNullRef() bool {
	return bool(C.WasmEdge_ValueIsNullRef(v.raw))
}

// ExternRef returns the pinned Go payload carried by an externref value.
// The result is a borrowed view: closing it is a no-op; the pin is released
// by closing the *ExternRef that created the value. It panics if the value
// is not an externref, and returns nil for a null externref or for an
// externref that was not produced by this package.
func (v Value) ExternRef() *ExternRef {
	v.mustKind(ValKindExternRef)
	p := C.WasmEdge_ValueGetExternRef(v.raw)
	if p == nil {
		return nil
	}
	return borrowedExternRef(uintptr(C.wasmedgego_ptrToHandle(p)))
}

// String renders the value for debugging.
func (v Value) String() string {
	switch v.Kind() {
	case ValKindI32:
		return fmt.Sprintf("i32:%d", v.I32())
	case ValKindI64:
		return fmt.Sprintf("i64:%d", v.I64())
	case ValKindF32:
		return fmt.Sprintf("f32:%g", v.F32())
	case ValKindF64:
		return fmt.Sprintf("f64:%g", v.F64())
	case ValKindV128:
		return fmt.Sprintf("v128:%x", v.V128())
	default:
		if v.IsNullRef() {
			return v.Kind().String() + ":null"
		}
		return v.Kind().String()
	}
}

// packValues copies vals into a C-layout slice. Passing &out[0] to a C call
// is safe: the slice holds no Go pointers (externref payloads are opaque
// handles, funcrefs are C pointers).
func packValues(vals []Value) []C.WasmEdge_Value {
	if len(vals) == 0 {
		return nil
	}
	out := make([]C.WasmEdge_Value, len(vals))
	for i, v := range vals {
		out[i] = v.raw
	}
	return out
}

// unpackValues wraps a C-filled slice back into Values.
func unpackValues(cvals []C.WasmEdge_Value) []Value {
	if len(cvals) == 0 {
		return nil
	}
	out := make([]Value, len(cvals))
	for i, cv := range cvals {
		out[i] = Value{raw: cv}
	}
	return out
}

// valuesPtr returns the C array head for a packed slice (nil for empty).
func valuesPtr(cvals []C.WasmEdge_Value) *C.WasmEdge_Value {
	if len(cvals) == 0 {
		return nil
	}
	return &cvals[0]
}

// NOTE(phase 4): FuncRefValue(*Function) and (Value).FuncRef() are added in
// function.go together with the Function type.
