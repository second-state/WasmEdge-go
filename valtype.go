package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

// ValType describes one WASM value type. Since WasmEdge 0.14 this is a
// small struct rather than an enum (the GC proposal's typed references do
// not fit in an enum), so ValType wraps the C struct by value and has no
// lifetime to manage.
type ValType struct {
	raw C.WasmEdge_ValType
}

// ValTypeI32 returns the i32 value type.
func ValTypeI32() ValType { return ValType{raw: C.WasmEdge_ValTypeGenI32()} }

// ValTypeI64 returns the i64 value type.
func ValTypeI64() ValType { return ValType{raw: C.WasmEdge_ValTypeGenI64()} }

// ValTypeF32 returns the f32 value type.
func ValTypeF32() ValType { return ValType{raw: C.WasmEdge_ValTypeGenF32()} }

// ValTypeF64 returns the f64 value type.
func ValTypeF64() ValType { return ValType{raw: C.WasmEdge_ValTypeGenF64()} }

// ValTypeV128 returns the v128 vector type (SIMD proposal).
func ValTypeV128() ValType { return ValType{raw: C.WasmEdge_ValTypeGenV128()} }

// ValTypeFuncRef returns the funcref reference type.
func ValTypeFuncRef() ValType { return ValType{raw: C.WasmEdge_ValTypeGenFuncRef()} }

// ValTypeExternRef returns the externref reference type.
func ValTypeExternRef() ValType { return ValType{raw: C.WasmEdge_ValTypeGenExternRef()} }

// Equal reports whether two value types are identical.
func (t ValType) Equal(o ValType) bool {
	return bool(C.WasmEdge_ValTypeIsEqual(t.raw, o.raw))
}

// IsI32 reports whether the type is i32.
func (t ValType) IsI32() bool { return bool(C.WasmEdge_ValTypeIsI32(t.raw)) }

// IsI64 reports whether the type is i64.
func (t ValType) IsI64() bool { return bool(C.WasmEdge_ValTypeIsI64(t.raw)) }

// IsF32 reports whether the type is f32.
func (t ValType) IsF32() bool { return bool(C.WasmEdge_ValTypeIsF32(t.raw)) }

// IsF64 reports whether the type is f64.
func (t ValType) IsF64() bool { return bool(C.WasmEdge_ValTypeIsF64(t.raw)) }

// IsV128 reports whether the type is v128.
func (t ValType) IsV128() bool { return bool(C.WasmEdge_ValTypeIsV128(t.raw)) }

// IsFuncRef reports whether the type is funcref.
func (t ValType) IsFuncRef() bool { return bool(C.WasmEdge_ValTypeIsFuncRef(t.raw)) }

// IsExternRef reports whether the type is externref.
func (t ValType) IsExternRef() bool { return bool(C.WasmEdge_ValTypeIsExternRef(t.raw)) }

// IsRef reports whether the type is any reference type, nullable or not
// (includes the GC proposal's typed references).
func (t ValType) IsRef() bool { return bool(C.WasmEdge_ValTypeIsRef(t.raw)) }

// IsRefNull reports whether the type is a nullable reference type.
func (t ValType) IsRefNull() bool { return bool(C.WasmEdge_ValTypeIsRefNull(t.raw)) }

// ValKind is a coarse classification of a ValType, convenient for switch
// statements. GC-proposal typed references that are none of the classic
// kinds report ValKindRef.
type ValKind uint8

const (
	ValKindUnknown ValKind = iota
	ValKindI32
	ValKindI64
	ValKindF32
	ValKindF64
	ValKindV128
	ValKindFuncRef
	ValKindExternRef
	ValKindRef // other reference types (GC proposal)
)

// Kind classifies the value type.
func (t ValType) Kind() ValKind {
	switch {
	case t.IsI32():
		return ValKindI32
	case t.IsI64():
		return ValKindI64
	case t.IsF32():
		return ValKindF32
	case t.IsF64():
		return ValKindF64
	case t.IsV128():
		return ValKindV128
	case t.IsFuncRef():
		return ValKindFuncRef
	case t.IsExternRef():
		return ValKindExternRef
	case t.IsRef():
		return ValKindRef
	default:
		return ValKindUnknown
	}
}

func (k ValKind) String() string {
	switch k {
	case ValKindI32:
		return "i32"
	case ValKindI64:
		return "i64"
	case ValKindF32:
		return "f32"
	case ValKindF64:
		return "f64"
	case ValKindV128:
		return "v128"
	case ValKindFuncRef:
		return "funcref"
	case ValKindExternRef:
		return "externref"
	case ValKindRef:
		return "ref"
	default:
		return "unknown"
	}
}

func (t ValType) String() string { return t.Kind().String() }
