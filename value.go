package wasmedge

// #include "shims.h"
import "C"

import (
	"encoding/binary"
	"fmt"
	"unsafe"
)

func newOpaqueToken() unsafe.Pointer {
	return C.wasmedgego_tokenCreate()
}

func deleteOpaqueToken(token unsafe.Pointer) {
	C.wasmedgego_tokenDelete(token)
}

// Value is one WASM value together with its type. Values are copied freely.
// Reference values remember the Go wrapper that owns their payload so
// containers and executions can validate and lease it. The caller must still
// keep that owner open while the value can be observed.
//
// Accessors panic when the value has a different kind, mirroring the
// reflect package's contract: a kind mismatch is a programmer error, not a
// runtime condition to handle.
type Value struct {
	raw   C.WasmEdge_Value
	owner any
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
// order (lane 0 first), independent of the host's native byte order.
func V128(lanes [16]byte) Value {
	return Value{raw: C.wasmedgego_valueGenV128(
		C.uint64_t(binary.LittleEndian.Uint64(lanes[:8])),
		C.uint64_t(binary.LittleEndian.Uint64(lanes[8:])),
	)}
}

// ExternRefValue returns an externref WASM value carrying r. The *ExternRef
// must stay open while the engine can still observe the value.
func ExternRefValue(r *ExternRef) Value {
	if r == nil {
		return NullExternRef()
	}
	r.assertAlive()
	return externRefValueFromToken(r.token(), r.state)
}

// externRefValueFromToken is the single raw-token constructor. Keeping it
// separate lets tests prove that unregistered and cross-domain C pointers
// are rejected without exposing unsafe pointer conversion in the public API.
func externRefValueFromToken(token unsafe.Pointer, owner any) Value {
	return Value{raw: C.WasmEdge_ValueGenExternRef(token), owner: owner}
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

func (v Value) assertOwnerAlive() {
	if owner, ok := v.owner.(aliveGuard); ok {
		owner.assertAlive()
	}
}

func assertValueOwnersAlive(values []Value) {
	for _, value := range values {
		value.assertOwnerAlive()
	}
}

// validateInvocationParams rejects failures the Go binding can prove before
// retaining reference arguments. A guest may persist a reference in a table
// or global once execution begins, so successful preflight is intentionally
// followed by conservative generation-long rooting.
//
// WasmEdge's full indexed-reference subtyping check requires the defining
// module's private type graph. We reject disjoint classic externref/funcref
// kinds and nullability here, while leaving other non-identical reference
// types to the engine.
func validateInvocationParams(ft FunctionType, values []Value) error {
	if len(values) != len(ft.Params) {
		return &Error{
			Category: ErrCategoryWASM,
			Code:     ErrCodeFuncSigMismatch,
			Message: fmt.Sprintf(
				"function wants %d parameters, got %d",
				len(ft.Params), len(values),
			),
		}
	}
	for i, value := range values {
		got, want := value.Type(), ft.Params[i]
		if got.Equal(want) {
			if want.IsRef() && !want.IsRefNull() && value.IsNullRef() {
				return &Error{
					Category: ErrCategoryWASM,
					Code:     ErrCodeNonNullRequired,
					Message: fmt.Sprintf(
						"function parameter %d requires a non-null %s",
						i, want,
					),
				}
			}
			continue
		}
		if got.IsRef() && want.IsRef() {
			// WasmEdge's private type graph is needed for indexed GC
			// references, but classic externref and funcref are provably
			// disjoint without it. Reject that mismatch before any argument
			// owner is retained.
			if got.Kind() != ValKindRef && want.Kind() != ValKindRef &&
				got.Kind() != want.Kind() {
				return &Error{
					Category: ErrCategoryWASM,
					Code:     ErrCodeFuncSigMismatch,
					Message: fmt.Sprintf(
						"function parameter %d has type %s, want %s",
						i, got, want,
					),
				}
			}
			if !want.IsRefNull() && value.IsNullRef() {
				return &Error{
					Category: ErrCategoryWASM,
					Code:     ErrCodeNonNullRequired,
					Message: fmt.Sprintf(
						"function parameter %d requires a non-null %s",
						i, want,
					),
				}
			}
			continue
		}
		return &Error{
			Category: ErrCategoryWASM,
			Code:     ErrCodeFuncSigMismatch,
			Message: fmt.Sprintf(
				"function parameter %d has type %s, want %s",
				i, got, want,
			),
		}
	}
	return nil
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
	var low, high C.uint64_t
	C.wasmedgego_valueGetV128(v.raw, &low, &high)
	var lanes [16]byte
	binary.LittleEndian.PutUint64(lanes[:8], uint64(low))
	binary.LittleEndian.PutUint64(lanes[8:], uint64(high))
	return lanes
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
	v.assertOwnerAlive()
	p := C.WasmEdge_ValueGetExternRef(v.raw)
	if p == nil {
		return nil
	}
	return borrowedExternRef(p, v.owner)
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

// packValues copies vals into a C-layout slice and validates the uint32 count
// used by every native invocation. Passing &out[0] to a C call is safe: the
// slice holds no Go pointers (externref payloads are opaque C-token pointers,
// funcrefs are C pointers).
func packValues(vals []Value) ([]C.WasmEdge_Value, C.uint32_t, error) {
	count, err := checkedUint32Count("function parameter", uint64(len(vals)))
	if err != nil {
		return nil, 0, err
	}
	if len(vals) == 0 {
		return nil, 0, nil
	}
	out := make([]C.WasmEdge_Value, len(vals))
	for i, v := range vals {
		out[i] = v.raw
	}
	return out, C.uint32_t(count), nil
}

// unpackValues wraps a C-filled slice back into Values.
func unpackValues(cvals []C.WasmEdge_Value) []Value {
	return unpackValuesOwned(cvals, nil)
}

// unpackValuesOwned restores reference provenance when possible. Externrefs
// resolve their package-owned opaque token to the canonical state; funcrefs
// reuse the owner of an identical source value; native-only references fall
// back to owner. Numeric values remain plain copies.
func unpackValuesOwned(cvals []C.WasmEdge_Value, owner any, sources ...Value) []Value {
	if len(cvals) == 0 {
		return nil
	}
	out := make([]Value, len(cvals))
	for i, cv := range cvals {
		out[i] = Value{raw: cv}
		if !out[i].Type().IsRef() || out[i].IsNullRef() {
			continue
		}
		if out[i].Kind() == ValKindExternRef {
			token := C.WasmEdge_ValueGetExternRef(cv)
			if state, ok := externRefTokens.Load(token); ok {
				out[i].owner = state
				continue
			}
		}
		for _, source := range sources {
			if sameReference(out[i], source) {
				out[i].owner = source.owner
				break
			}
		}
		if out[i].owner == nil {
			out[i].owner = owner
		}
	}
	return out
}

func sameReference(left, right Value) bool {
	if left.Kind() != right.Kind() || !left.Type().IsRef() || right.IsNullRef() {
		return false
	}
	switch left.Kind() {
	case ValKindExternRef:
		return C.WasmEdge_ValueGetExternRef(left.raw) == C.WasmEdge_ValueGetExternRef(right.raw)
	case ValKindFuncRef:
		return C.WasmEdge_ValueGetFuncRef(left.raw) == C.WasmEdge_ValueGetFuncRef(right.raw)
	default:
		return false
	}
}

func valueOwners(values []Value) []any {
	owners := make([]any, 0, len(values))
	for _, value := range values {
		if value.Type().IsRef() && value.owner != nil {
			owners = append(owners, value.owner)
		}
	}
	return owners
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
