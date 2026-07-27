package wasmedge

import "fmt"

// ComponentTypeCode identifies a Component Model interface type.
//
// WasmEdge 0.17.1 exposes these codes for type inspection, but does not
// expose a stable Component Model runtime API. This type mirrors only the
// WasmEdge_ComponentTypeCode enumeration.
type ComponentTypeCode uint8

// ComponentTypeCode values mirror WasmEdge 0.17.1's
// WasmEdge_ComponentTypeCode enumeration.
const (
	ComponentTypeCodeTypeIndex  ComponentTypeCode = 0x00
	ComponentTypeCodeBool       ComponentTypeCode = 0x7F
	ComponentTypeCodeS8         ComponentTypeCode = 0x7E
	ComponentTypeCodeU8         ComponentTypeCode = 0x7D
	ComponentTypeCodeS16        ComponentTypeCode = 0x7C
	ComponentTypeCodeU16        ComponentTypeCode = 0x7B
	ComponentTypeCodeS32        ComponentTypeCode = 0x7A
	ComponentTypeCodeU32        ComponentTypeCode = 0x79
	ComponentTypeCodeS64        ComponentTypeCode = 0x78
	ComponentTypeCodeU64        ComponentTypeCode = 0x77
	ComponentTypeCodeF32        ComponentTypeCode = 0x76
	ComponentTypeCodeF64        ComponentTypeCode = 0x75
	ComponentTypeCodeChar       ComponentTypeCode = 0x74
	ComponentTypeCodeString     ComponentTypeCode = 0x73
	ComponentTypeCodeRecord     ComponentTypeCode = 0x72
	ComponentTypeCodeVariant    ComponentTypeCode = 0x71
	ComponentTypeCodeList       ComponentTypeCode = 0x70
	ComponentTypeCodeTuple      ComponentTypeCode = 0x6F
	ComponentTypeCodeFlags      ComponentTypeCode = 0x6E
	ComponentTypeCodeEnum       ComponentTypeCode = 0x6D
	ComponentTypeCodeOption     ComponentTypeCode = 0x6B
	ComponentTypeCodeResult     ComponentTypeCode = 0x6A
	ComponentTypeCodeOwn        ComponentTypeCode = 0x69
	ComponentTypeCodeBorrow     ComponentTypeCode = 0x68
	ComponentTypeCodeListLen    ComponentTypeCode = 0x67
	ComponentTypeCodeStream     ComponentTypeCode = 0x66
	ComponentTypeCodeFuture     ComponentTypeCode = 0x65
	ComponentTypeCodeErrContext ComponentTypeCode = 0x64
)

// String returns the diagnostic text assigned to c by WasmEdge 0.17.1.
func (c ComponentTypeCode) String() string {
	switch c {
	case ComponentTypeCodeTypeIndex:
		return "type_index"
	case ComponentTypeCodeBool:
		return "bool"
	case ComponentTypeCodeS8:
		return "s8"
	case ComponentTypeCodeU8:
		return "u8"
	case ComponentTypeCodeS16:
		return "s16"
	case ComponentTypeCodeU16:
		return "u16"
	case ComponentTypeCodeS32:
		return "s32"
	case ComponentTypeCodeU32:
		return "u32"
	case ComponentTypeCodeS64:
		return "s64"
	case ComponentTypeCodeU64:
		return "u64"
	case ComponentTypeCodeF32:
		return "f32"
	case ComponentTypeCodeF64:
		return "f64"
	case ComponentTypeCodeChar:
		return "char"
	case ComponentTypeCodeString:
		return "string"
	case ComponentTypeCodeRecord:
		return "record"
	case ComponentTypeCodeVariant:
		return "variant"
	case ComponentTypeCodeList:
		return "list"
	case ComponentTypeCodeTuple:
		return "tuple"
	case ComponentTypeCodeFlags:
		return "flags"
	case ComponentTypeCodeEnum:
		return "enum"
	case ComponentTypeCodeOption:
		// This is the spelling in WasmEdge 0.17.1's enum.inc.
		return "enum"
	case ComponentTypeCodeResult:
		return "result"
	case ComponentTypeCodeOwn:
		return "own"
	case ComponentTypeCodeBorrow:
		// This is the spelling in WasmEdge 0.17.1's enum.inc.
		return "own"
	case ComponentTypeCodeListLen:
		return "list_len"
	case ComponentTypeCodeStream:
		return "stream"
	case ComponentTypeCodeFuture:
		return "future"
	case ComponentTypeCodeErrContext:
		return "error-context"
	default:
		return fmt.Sprintf("component type code 0x%02x", uint8(c))
	}
}
