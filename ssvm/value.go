package ssvm

// #include <ssvm.h>
import "C"
import "unsafe"

type ValType C.enum_SSVM_ValType
type RefType C.enum_SSVM_RefType
type ValMut C.enum_SSVM_Mutability

const (
	ValType_I32       = ValType(C.SSVM_ValType_I32)
	ValType_I64       = ValType(C.SSVM_ValType_I64)
	ValType_F32       = ValType(C.SSVM_ValType_F32)
	ValType_F64       = ValType(C.SSVM_ValType_F64)
	ValType_V128      = ValType(C.SSVM_ValType_V128)
	ValType_FuncRef   = ValType(C.SSVM_ValType_FuncRef)
	ValType_ExternRef = ValType(C.SSVM_ValType_ExternRef)
)

const (
	RefType_FuncRef   = RefType(C.SSVM_ValType_FuncRef)
	RefType_ExternRef = RefType(C.SSVM_ValType_ExternRef)
)

const (
	ValMut_Const = ValMut(C.SSVM_Mutability_Const)
	ValMut_Var   = ValMut(C.SSVM_Mutability_Var)
)

func (self ValType) String() string {
	switch self {
	case ValType_I32:
		return "i32"
	case ValType_I64:
		return "i64"
	case ValType_F32:
		return "f32"
	case ValType_F64:
		return "f64"
	case ValType_V128:
		return "v128"
	case ValType_FuncRef:
		return "funcref"
	case ValType_ExternRef:
		return "externref"
	}
	panic("Unknown value type")
}

func (self RefType) String() string {
	switch self {
	case RefType_FuncRef:
		return "funcref"
	case RefType_ExternRef:
		return "externref"
	}
	panic("Unknown reference type")
}

func (self ValMut) String() string {
	switch self {
	case ValMut_Const:
		return "const"
	case ValMut_Var:
		return "var"
	}
	panic("Unknown value mutability")
}

type FuncRef struct {
	_inner C.SSVM_Value
}

type ExternRef struct {
	_inner C.SSVM_Value
}

type V128 struct {
	_inner C.SSVM_Value
}

func toSSVMValue(value interface{}) C.SSVM_Value {
	switch value.(type) {
	case FuncRef:
		return value.(FuncRef)._inner
	case ExternRef:
		return value.(ExternRef)._inner
	case V128:
		return value.(V128)._inner
	case int:
		if unsafe.Sizeof(value.(int)) == 4 {
			return C.SSVM_ValueGenI32(C.int32_t(value.(int)))
		} else {
			return C.SSVM_ValueGenI64(C.int64_t(value.(int)))
		}
	case int32:
		return C.SSVM_ValueGenI32(C.int32_t(value.(int32)))
	case int64:
		return C.SSVM_ValueGenI64(C.int64_t(value.(int64)))
	case uint:
		if unsafe.Sizeof(value.(uint)) == 4 {
			return C.SSVM_ValueGenI32(C.int32_t(int32(value.(uint))))
		} else {
			return C.SSVM_ValueGenI64(C.int64_t(int64(value.(uint))))
		}
	case uint32:
		return C.SSVM_ValueGenI32(C.int32_t(int32(value.(uint32))))
	case uint64:
		return C.SSVM_ValueGenI64(C.int64_t(int64(value.(uint64))))
	case float32:
		return C.SSVM_ValueGenF32(C.float(value.(float32)))
	case float64:
		return C.SSVM_ValueGenF64(C.double(value.(float64)))
	default:
		panic("Wrong argument of toSSVMValue()")
	}
}

func fromSSVMValue(value C.SSVM_Value, origtype C.enum_SSVM_ValType) interface{} {
	switch origtype {
	case C.SSVM_ValType_I32:
		return int32(C.SSVM_ValueGetI32(value))
	case C.SSVM_ValType_I64:
		return int64(C.SSVM_ValueGetI64(value))
	case C.SSVM_ValType_F32:
		return float32(C.SSVM_ValueGetF32(value))
	case C.SSVM_ValType_F64:
		return float64(C.SSVM_ValueGetF64(value))
	case C.SSVM_ValType_V128:
		return V128{_inner: value}
	case C.SSVM_ValType_FuncRef:
		return FuncRef{_inner: value}
	case C.SSVM_ValType_ExternRef:
		return ExternRef{_inner: value}
	default:
		panic("Wrong argument of fromSSVMValue()")
	}
	return 0
}

func toSSVMValueSlide(vals ...interface{}) []C.SSVM_Value {
	cvals := make([]C.SSVM_Value, len(vals))
	for i, val := range vals {
		cvals[i] = toSSVMValue(val)
	}
	return cvals
}

func fromSSVMValueSlide(cvals []C.SSVM_Value, types []C.enum_SSVM_ValType) []interface{} {
	if len(types) > 0 {
		vals := make([]interface{}, len(types))
		for i, cval := range cvals {
			vals[i] = fromSSVMValue(cval, types[i])
		}
		return vals
	}
	return []interface{}{}
}
