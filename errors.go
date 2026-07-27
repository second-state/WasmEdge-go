package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"errors"
	"fmt"
	"sync"
)

// ErrCategory identifies who produced an error code: the WASM runtime or the
// embedder (host functions returning custom errors).
type ErrCategory uint32

// ErrCategory values distinguish runtime errors from embedder-defined errors.
const (
	ErrCategoryWASM      ErrCategory = C.WasmEdge_ErrCategory_WASM
	ErrCategoryUserLevel ErrCategory = C.WasmEdge_ErrCategory_UserLevelError
)

func (c ErrCategory) String() string {
	switch c {
	case ErrCategoryWASM:
		return "wasm"
	case ErrCategoryUserLevel:
		return "user"
	default:
		return fmt.Sprintf("category(%d)", uint32(c))
	}
}

// ErrCode is a 24-bit WasmEdge error code. For ErrCategoryWASM the values
// are defined by the runtime; for ErrCategoryUserLevel they are defined by
// the embedder.
type ErrCode uint32

// ErrCode values mirror every entry in the UseErrCode table from WasmEdge
// 0.17.1's include/common/enum.inc. The high byte identifies the phase; the
// 0xA0 sub-ranges contain component-model errors.
const (
	ErrCodeSuccess             ErrCode = 0x0000
	ErrCodeTerminated          ErrCode = 0x0001
	ErrCodeRuntimeError        ErrCode = 0x0002
	ErrCodeCostLimitExceeded   ErrCode = 0x0003
	ErrCodeWrongVMWorkflow     ErrCode = 0x0004
	ErrCodeFuncNotFound        ErrCode = 0x0005
	ErrCodeAOTDisabled         ErrCode = 0x0006
	ErrCodeInterrupted         ErrCode = 0x0007
	ErrCodeNotValidated        ErrCode = 0x0008
	ErrCodeNonNullRequired     ErrCode = 0x0009
	ErrCodeSetValueToConst     ErrCode = 0x000A
	ErrCodeSetValueErrorType   ErrCode = 0x000B
	ErrCodeUserDefError        ErrCode = 0x000C
	ErrCodeInvalidAOTConfigure ErrCode = 0x000D
	ErrCodeAOTNotImpl          ErrCode = 0x000E

	ErrCodeIllegalPath                   ErrCode = 0x0100
	ErrCodeReadError                     ErrCode = 0x0101
	ErrCodeUnexpectedEnd                 ErrCode = 0x0102
	ErrCodeMalformedMagic                ErrCode = 0x0103
	ErrCodeMalformedVersion              ErrCode = 0x0104
	ErrCodeMalformedSection              ErrCode = 0x0105
	ErrCodeSectionSizeMismatch           ErrCode = 0x0106
	ErrCodeLengthOutOfBounds             ErrCode = 0x0107
	ErrCodeJunkSection                   ErrCode = 0x0108
	ErrCodeIncompatibleFuncCode          ErrCode = 0x0109
	ErrCodeIncompatibleDataCount         ErrCode = 0x010A
	ErrCodeDataCountRequired             ErrCode = 0x010B
	ErrCodeMalformedImportKind           ErrCode = 0x010C
	ErrCodeMalformedExportKind           ErrCode = 0x010D
	ErrCodeExpectedZeroByte              ErrCode = 0x010E
	ErrCodeInvalidMut                    ErrCode = 0x010F
	ErrCodeTooManyLocals                 ErrCode = 0x0110
	ErrCodeMalformedValType              ErrCode = 0x0111
	ErrCodeMalformedElemType             ErrCode = 0x0112
	ErrCodeMalformedRefType              ErrCode = 0x0113
	ErrCodeMalformedUTF8                 ErrCode = 0x0114
	ErrCodeIntegerTooLarge               ErrCode = 0x0115
	ErrCodeIntegerTooLong                ErrCode = 0x0116
	ErrCodeIllegalOpCode                 ErrCode = 0x0117
	ErrCodeENDCodeExpected               ErrCode = 0x0118
	ErrCodeIllegalGrammar                ErrCode = 0x0119
	ErrCodeIntrinsicsTableNotFound       ErrCode = 0x011A
	ErrCodeMalformedTable                ErrCode = 0x011B
	ErrCodeMalformedMemoryOpFlags        ErrCode = 0x011C
	ErrCodeMalformedLimitFlags           ErrCode = 0x011D
	ErrCodeMalformedCatchFlags           ErrCode = 0x011E
	ErrCodeMalformedSort                 ErrCode = 0x01A0
	ErrCodeMalformedAliasTarget          ErrCode = 0x01A1
	ErrCodeMalformedCoreInstance         ErrCode = 0x01A2
	ErrCodeMalformedInstance             ErrCode = 0x01A3
	ErrCodeMalformedDefType              ErrCode = 0x01A4
	ErrCodeMalformedModuleType           ErrCode = 0x01A5
	ErrCodeMalformedRecordType           ErrCode = 0x01A6
	ErrCodeMalformedVariantType          ErrCode = 0x01A7
	ErrCodeMalformedTupleType            ErrCode = 0x01A8
	ErrCodeMalformedFlagsType            ErrCode = 0x01A9
	ErrCodeMalformedCanonical            ErrCode = 0x01AA
	ErrCodeUnknownCanonicalOption        ErrCode = 0x01AB
	ErrCodeMalformedName                 ErrCode = 0x01AC
	ErrCodeComponentNotImplLoader        ErrCode = 0x01AD
	ErrCodeInvalidAlignment              ErrCode = 0x0200
	ErrCodeAlignmentTooLarge             ErrCode = 0x0201
	ErrCodeTypeCheckFailed               ErrCode = 0x0202
	ErrCodeInvalidLabelIdx               ErrCode = 0x0203
	ErrCodeInvalidLocalIdx               ErrCode = 0x0204
	ErrCodeInvalidFieldIdx               ErrCode = 0x0205
	ErrCodeInvalidFuncTypeIdx            ErrCode = 0x0206
	ErrCodeInvalidFuncIdx                ErrCode = 0x0207
	ErrCodeInvalidTableIdx               ErrCode = 0x0208
	ErrCodeInvalidMemoryIdx              ErrCode = 0x0209
	ErrCodeInvalidGlobalIdx              ErrCode = 0x020A
	ErrCodeInvalidTagIdx                 ErrCode = 0x020B
	ErrCodeInvalidElemIdx                ErrCode = 0x020C
	ErrCodeInvalidDataIdx                ErrCode = 0x020D
	ErrCodeInvalidRefIdx                 ErrCode = 0x020E
	ErrCodeConstExprRequired             ErrCode = 0x020F
	ErrCodeDupExportName                 ErrCode = 0x0210
	ErrCodeImmutableGlobal               ErrCode = 0x0211
	ErrCodeImmutableField                ErrCode = 0x0212
	ErrCodeImmutableArray                ErrCode = 0x0213
	ErrCodeInvalidResultArity            ErrCode = 0x0214
	ErrCodeMultiTables                   ErrCode = 0x0215
	ErrCodeMultiMemories                 ErrCode = 0x0216
	ErrCodeInvalidLimit                  ErrCode = 0x0217
	ErrCodeInvalidMemPages               ErrCode = 0x0218
	ErrCodeInvalidStartFunc              ErrCode = 0x0219
	ErrCodeInvalidLaneIdx                ErrCode = 0x021A
	ErrCodeInvalidUninitLocal            ErrCode = 0x021B
	ErrCodeInvalidNotDefaultableField    ErrCode = 0x021C
	ErrCodeInvalidNotDefaultableArray    ErrCode = 0x021D
	ErrCodeInvalidPackedField            ErrCode = 0x021E
	ErrCodeInvalidPackedArray            ErrCode = 0x021F
	ErrCodeInvalidUnpackedField          ErrCode = 0x0220
	ErrCodeInvalidUnpackedArray          ErrCode = 0x0221
	ErrCodeInvalidBrRefType              ErrCode = 0x0222
	ErrCodeArrayTypesMismatch            ErrCode = 0x0223
	ErrCodeArrayTypesNumtypeRequired     ErrCode = 0x0224
	ErrCodeInvalidSubType                ErrCode = 0x0225
	ErrCodeInvalidTagResultType          ErrCode = 0x0226
	ErrCodeSharedMemoryNoMax             ErrCode = 0x0227
	ErrCodeInvalidMemPages64             ErrCode = 0x0228
	ErrCodeInvalidTableSize64            ErrCode = 0x0229
	ErrCodeInvalidOffset                 ErrCode = 0x022A
	ErrCodeMissingArgument               ErrCode = 0x02A0
	ErrCodeArgTypeMismatch               ErrCode = 0x02A1
	ErrCodeInvalidIndex                  ErrCode = 0x02A2
	ErrCodeInvalidTypeReference          ErrCode = 0x02A3
	ErrCodeExportNotFound                ErrCode = 0x02A4
	ErrCodeComponentNotImplValidator     ErrCode = 0x02A5
	ErrCodeComponentDuplicateName        ErrCode = 0x02A6
	ErrCodeComponentInvalidName          ErrCode = 0x02A7
	ErrCodeNotADefinedType               ErrCode = 0x02A8
	ErrCodeDefTypeIndexOutOfBounds       ErrCode = 0x02A9
	ErrCodeNameCannotBeEmpty             ErrCode = 0x02AA
	ErrCodeCannotHaveMoreThan32Flags     ErrCode = 0x02AB
	ErrCodeRecordFieldNameConflicts      ErrCode = 0x02AC
	ErrCodeVariantCaseNameConflicts      ErrCode = 0x02AD
	ErrCodeFlagNameConflicts             ErrCode = 0x02AE
	ErrCodeEnumTagNameConflicts          ErrCode = 0x02AF
	ErrCodeVariantMustHaveCase           ErrCode = 0x02B0
	ErrCodeExportAscriptionIncompatible  ErrCode = 0x02B1
	ErrCodeInstanceMissingExpectedExport ErrCode = 0x02B2
	ErrCodeInvalidExportName             ErrCode = 0x02B3
	ErrCodeInvalidExternName             ErrCode = 0x02B4
	ErrCodeModuleNameConflict            ErrCode = 0x0300
	ErrCodeIncompatibleImportType        ErrCode = 0x0301
	ErrCodeUnknownImport                 ErrCode = 0x0302
	ErrCodeDataSegDoesNotFit             ErrCode = 0x0303
	ErrCodeElemSegDoesNotFit             ErrCode = 0x0304
	ErrCodeInvalidCoreSort               ErrCode = 0x03A0
	ErrCodeInvalidCanonOption            ErrCode = 0x03A1
	ErrCodeCoreInvalidExport             ErrCode = 0x03A2
	ErrCodeResourceDropArgument          ErrCode = 0x03A3
	ErrCodeComponentNotImplInstantiate   ErrCode = 0x03A4
	ErrCodeWrongInstanceAddress          ErrCode = 0x0400
	ErrCodeWrongInstanceIndex            ErrCode = 0x0401
	ErrCodeInstrTypeMismatch             ErrCode = 0x0402
	ErrCodeFuncSigMismatch               ErrCode = 0x0403
	ErrCodeDivideByZero                  ErrCode = 0x0404
	ErrCodeIntegerOverflow               ErrCode = 0x0405
	ErrCodeInvalidConvToInt              ErrCode = 0x0406
	ErrCodeTableOutOfBounds              ErrCode = 0x0407
	ErrCodeMemoryOutOfBounds             ErrCode = 0x0408
	ErrCodeArrayOutOfBounds              ErrCode = 0x0409
	ErrCodeUnreachable                   ErrCode = 0x040A
	ErrCodeUninitializedElement          ErrCode = 0x040B
	ErrCodeUndefinedElement              ErrCode = 0x040C
	ErrCodeIndirectCallTypeMismatch      ErrCode = 0x040D
	ErrCodeHostFuncError                 ErrCode = 0x040E
	ErrCodeRefTypeMismatch               ErrCode = 0x040F
	ErrCodeUnalignedAtomicAccess         ErrCode = 0x0410
	ErrCodeExpectSharedMemory            ErrCode = 0x0411
	ErrCodeCastNullToNonNull             ErrCode = 0x0412
	ErrCodeAccessNullFunc                ErrCode = 0x0413
	ErrCodeAccessNullStruct              ErrCode = 0x0414
	ErrCodeAccessNullArray               ErrCode = 0x0415
	ErrCodeAccessNullI31                 ErrCode = 0x0416
	ErrCodeAccessNullException           ErrCode = 0x0417
	ErrCodeCastFailed                    ErrCode = 0x0418
	ErrCodeUncaughtException             ErrCode = 0x0419
)

// String returns the diagnostic text assigned to c by WasmEdge 0.17.1. It
// includes the numeric value for unknown codes so callers remain debuggable
// when running against a newer compatible patch release.
func (c ErrCode) String() string {
	switch c {
	case ErrCodeSuccess:
		return "success"
	case ErrCodeTerminated:
		return "terminated"
	case ErrCodeRuntimeError:
		return "generic runtime error"
	case ErrCodeCostLimitExceeded:
		return "cost limit exceeded"
	case ErrCodeWrongVMWorkflow:
		return "wrong VM workflow"
	case ErrCodeFuncNotFound:
		return "wasm function not found"
	case ErrCodeAOTDisabled:
		return "AOT runtime is disabled in this build"
	case ErrCodeInterrupted:
		return "execution interrupted"
	case ErrCodeNotValidated:
		return "wasm module hasn't passed validation yet"
	case ErrCodeNonNullRequired:
		return "set null value into non-nullable value type"
	case ErrCodeSetValueToConst:
		return "set value into const"
	case ErrCodeSetValueErrorType:
		return "set value type mismatch"
	case ErrCodeUserDefError:
		return "user defined error code"
	case ErrCodeInvalidAOTConfigure:
		return "invalid AOT/JIT configure"
	case ErrCodeAOTNotImpl:
		return "Not implemented instructions in AOT/JIT"
	case ErrCodeIllegalPath:
		return "invalid path"
	case ErrCodeReadError:
		return "read error"
	case ErrCodeUnexpectedEnd:
		return "unexpected end"
	case ErrCodeMalformedMagic:
		return "magic header not detected"
	case ErrCodeMalformedVersion:
		return "unknown binary version"
	case ErrCodeMalformedSection:
		return "malformed section id"
	case ErrCodeSectionSizeMismatch:
		return "section size mismatch"
	case ErrCodeLengthOutOfBounds:
		return "length out of bounds"
	case ErrCodeJunkSection:
		return "unexpected content after last section"
	case ErrCodeIncompatibleFuncCode:
		return "function and code section have inconsistent lengths"
	case ErrCodeIncompatibleDataCount:
		return "data count and data section have inconsistent lengths"
	case ErrCodeDataCountRequired:
		return "data count section required"
	case ErrCodeMalformedImportKind:
		return "malformed import kind"
	case ErrCodeMalformedExportKind:
		return "malformed export kind"
	case ErrCodeExpectedZeroByte:
		return "zero byte expected"
	case ErrCodeInvalidMut:
		return "malformed mutability"
	case ErrCodeTooManyLocals:
		return "too many locals"
	case ErrCodeMalformedValType:
		return "malformed value type"
	case ErrCodeMalformedElemType:
		return "malformed element type"
	case ErrCodeMalformedRefType:
		return "malformed reference type"
	case ErrCodeMalformedUTF8:
		return "malformed UTF-8 encoding"
	case ErrCodeIntegerTooLarge:
		return "integer too large"
	case ErrCodeIntegerTooLong:
		return "integer representation too long"
	case ErrCodeIllegalOpCode:
		return "illegal opcode"
	case ErrCodeENDCodeExpected:
		return "END opcode expected"
	case ErrCodeIllegalGrammar:
		return "invalid wasm grammar"
	case ErrCodeIntrinsicsTableNotFound:
		return "intrinsics table not found"
	case ErrCodeMalformedTable:
		return "malformed table"
	case ErrCodeMalformedMemoryOpFlags:
		return "malformed memop flags"
	case ErrCodeMalformedLimitFlags:
		return "malformed limits flags"
	case ErrCodeMalformedCatchFlags:
		return "malformed catch flags"
	case ErrCodeMalformedSort:
		return "malformed sort"
	case ErrCodeMalformedAliasTarget:
		return "malformed alias target"
	case ErrCodeMalformedCoreInstance:
		return "malformed core instance"
	case ErrCodeMalformedInstance:
		return "malformed instance"
	case ErrCodeMalformedDefType:
		return "malformed defined type"
	case ErrCodeMalformedModuleType:
		return "malformed module type"
	case ErrCodeMalformedRecordType:
		return "malformed record type"
	case ErrCodeMalformedVariantType:
		return "malformed variant type"
	case ErrCodeMalformedTupleType:
		return "malformed tuple type"
	case ErrCodeMalformedFlagsType:
		return "malformed flags type"
	case ErrCodeMalformedCanonical:
		return "malformed canonical"
	case ErrCodeUnknownCanonicalOption:
		return "unknown canonical option"
	case ErrCodeMalformedName:
		return "malformed name"
	case ErrCodeComponentNotImplLoader:
		return "component model (loader) not implemented"
	case ErrCodeInvalidAlignment:
		return "atomic alignment must be natural"
	case ErrCodeAlignmentTooLarge:
		return "alignment must not be larger than natural"
	case ErrCodeTypeCheckFailed:
		return "type mismatch"
	case ErrCodeInvalidLabelIdx:
		return "unknown label"
	case ErrCodeInvalidLocalIdx:
		return "unknown local"
	case ErrCodeInvalidFieldIdx:
		return "unknown field"
	case ErrCodeInvalidFuncTypeIdx:
		return "unknown type"
	case ErrCodeInvalidFuncIdx:
		return "unknown function"
	case ErrCodeInvalidTableIdx:
		return "unknown table"
	case ErrCodeInvalidMemoryIdx:
		return "unknown memory"
	case ErrCodeInvalidGlobalIdx:
		return "unknown global"
	case ErrCodeInvalidTagIdx:
		return "unknown tag"
	case ErrCodeInvalidElemIdx:
		return "unknown elem segment"
	case ErrCodeInvalidDataIdx:
		return "unknown data segment"
	case ErrCodeInvalidRefIdx:
		return "undeclared function reference"
	case ErrCodeConstExprRequired:
		return "constant expression required"
	case ErrCodeDupExportName:
		return "duplicate export name"
	case ErrCodeImmutableGlobal:
		return "immutable global"
	case ErrCodeImmutableField:
		return "immutable field"
	case ErrCodeImmutableArray:
		return "immutable array"
	case ErrCodeInvalidResultArity:
		return "invalid result arity"
	case ErrCodeMultiTables:
		return "multiple tables"
	case ErrCodeMultiMemories:
		return "multiple memories"
	case ErrCodeInvalidLimit:
		return "size minimum must not be greater than maximum"
	case ErrCodeInvalidMemPages:
		return "memory size must be at most 65536 pages (4GiB)"
	case ErrCodeInvalidStartFunc:
		return "start function"
	case ErrCodeInvalidLaneIdx:
		return "invalid lane index"
	case ErrCodeInvalidUninitLocal:
		return "uninitialized local"
	case ErrCodeInvalidNotDefaultableField:
		return "field type is not defaultable"
	case ErrCodeInvalidNotDefaultableArray:
		return "array type is not defaultable"
	case ErrCodeInvalidPackedField:
		return "field is packed"
	case ErrCodeInvalidPackedArray:
		return "array is packed"
	case ErrCodeInvalidUnpackedField:
		return "field is unpacked"
	case ErrCodeInvalidUnpackedArray:
		return "array is unpacked"
	case ErrCodeInvalidBrRefType:
		return "invalid br ref type"
	case ErrCodeArrayTypesMismatch:
		return "array types do not match"
	case ErrCodeArrayTypesNumtypeRequired:
		return "array type is not numeric or vector"
	case ErrCodeInvalidSubType:
		return "sub type"
	case ErrCodeInvalidTagResultType:
		return "non-empty tag result type"
	case ErrCodeSharedMemoryNoMax:
		return "shared memory must have maximum"
	case ErrCodeInvalidMemPages64:
		return "memory size"
	case ErrCodeInvalidTableSize64:
		return "table size"
	case ErrCodeInvalidOffset:
		return "offset out of range"
	case ErrCodeMissingArgument:
		return "missing argument"
	case ErrCodeArgTypeMismatch:
		return "type mismatch"
	case ErrCodeInvalidIndex:
		return "invalid index"
	case ErrCodeInvalidTypeReference:
		return "invalid type reference"
	case ErrCodeExportNotFound:
		return "export not found"
	case ErrCodeComponentNotImplValidator:
		return "component model (validator) not implemented"
	case ErrCodeComponentDuplicateName:
		return "duplicate name in component"
	case ErrCodeComponentInvalidName:
		return "invalid component name"
	case ErrCodeNotADefinedType:
		return "not a defined type"
	case ErrCodeDefTypeIndexOutOfBounds:
		return "index out of bounds"
	case ErrCodeNameCannotBeEmpty:
		return "name cannot be empty"
	case ErrCodeCannotHaveMoreThan32Flags:
		return "cannot have more than 32 flags"
	case ErrCodeRecordFieldNameConflicts:
		return "record field name conflicts with previous field name"
	case ErrCodeVariantCaseNameConflicts:
		return "variant case name conflicts with previous case name"
	case ErrCodeFlagNameConflicts:
		return "flag name conflicts with previous flag name"
	case ErrCodeEnumTagNameConflicts:
		return "enum tag name conflicts with previous tag name"
	case ErrCodeVariantMustHaveCase:
		return "variant type must have at least one case"
	case ErrCodeExportAscriptionIncompatible:
		return "ascribed type of export is not compatible"
	case ErrCodeInstanceMissingExpectedExport:
		return "missing expected export"
	case ErrCodeInvalidExportName:
		return "not a valid export name"
	case ErrCodeInvalidExternName:
		return "not a valid extern name"
	case ErrCodeModuleNameConflict:
		return "module name conflict"
	case ErrCodeIncompatibleImportType:
		return "incompatible import type"
	case ErrCodeUnknownImport:
		return "unknown import"
	case ErrCodeDataSegDoesNotFit:
		return "data segment does not fit"
	case ErrCodeElemSegDoesNotFit:
		return "elements segment does not fit"
	case ErrCodeInvalidCoreSort:
		return "invalid core sort"
	case ErrCodeInvalidCanonOption:
		return "invalid canonical option"
	case ErrCodeCoreInvalidExport:
		return "invalid export in core module"
	case ErrCodeResourceDropArgument:
		return "invalid argument been used to invoke resource.drop"
	case ErrCodeComponentNotImplInstantiate:
		return "component model (instantiation) not implemented"
	case ErrCodeWrongInstanceAddress:
		return "wrong instance address"
	case ErrCodeWrongInstanceIndex:
		return "wrong instance index"
	case ErrCodeInstrTypeMismatch:
		return "instruction type mismatch"
	case ErrCodeFuncSigMismatch:
		return "function signature mismatch"
	case ErrCodeDivideByZero:
		return "integer divide by zero"
	case ErrCodeIntegerOverflow:
		return "integer overflow"
	case ErrCodeInvalidConvToInt:
		return "invalid conversion to integer"
	case ErrCodeTableOutOfBounds:
		return "out of bounds table access"
	case ErrCodeMemoryOutOfBounds:
		return "out of bounds memory access"
	case ErrCodeArrayOutOfBounds:
		return "out of bounds array access"
	case ErrCodeUnreachable:
		return "unreachable"
	case ErrCodeUninitializedElement:
		return "uninitialized element"
	case ErrCodeUndefinedElement:
		return "undefined element"
	case ErrCodeIndirectCallTypeMismatch:
		return "indirect call type mismatch"
	case ErrCodeHostFuncError:
		return "host function failed"
	case ErrCodeRefTypeMismatch:
		return "reference type mismatch"
	case ErrCodeUnalignedAtomicAccess:
		return "unaligned atomic"
	case ErrCodeExpectSharedMemory:
		return "expected shared memory"
	case ErrCodeCastNullToNonNull:
		return "null reference"
	case ErrCodeAccessNullFunc:
		return "null function reference"
	case ErrCodeAccessNullStruct:
		return "null structure reference"
	case ErrCodeAccessNullArray:
		return "null array reference"
	case ErrCodeAccessNullI31:
		return "null i31 reference"
	case ErrCodeAccessNullException:
		return "null exception reference"
	case ErrCodeCastFailed:
		return "cast failure"
	case ErrCodeUncaughtException:
		return "uncaught exception"
	default:
		return fmt.Sprintf("error code 0x%04x", uint32(c))
	}
}

var (
	// ErrClosed reports an operation on a resource that has been closed or
	// whose ownership has already moved into another WasmEdge object.
	ErrClosed = errors.New("wasmedge: resource is closed")

	// ErrOwnership reports that an ownership-transferring operation received a
	// borrowed resource. Borrowed views may be inspected but never adopted.
	ErrOwnership = errors.New("wasmedge: resource is not owned by the caller")

	// ErrAlreadyExists reports an attempt to define an item under a name that
	// is already present in the target module.
	ErrAlreadyExists = errors.New("wasmedge: item already exists")
)

// HostPanicError reports a panic contained at a host-function cgo boundary.
// Stack is the Go stack captured at the panic site.
type HostPanicError struct {
	Value any
	Stack []byte
}

func (e *HostPanicError) Error() string {
	return fmt.Sprintf("wasmedge: host function panicked: %v", e.Value)
}

// Error is a failure reported by the WasmEdge runtime or by a host function.
// It satisfies the error interface, and errors.Is matches two *Error values
// by (Category, Code), so the sentinels below work with errors.Is even after
// the engine round-trips them.
type Error struct {
	Category ErrCategory
	Code     ErrCode
	// Message is the engine's description. It is captured when the error is
	// created; for ErrCategoryUserLevel codes the engine has no message and
	// this may be empty.
	Message string
}

func (e *Error) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "unknown error"
	}
	return fmt.Sprintf("wasmedge: %s (%s code 0x%04x)", msg, e.Category, uint32(e.Code))
}

// Is reports whether target is an *Error with the same category and code,
// enabling errors.Is(err, wasmedge.Terminate)-style comparisons.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && e.Category == t.Category && e.Code == t.Code
}

// Terminate is the graceful-stop sentinel. A host function that returns it
// stops WASM execution without reporting a failure, exactly like WASI
// proc_exit: the enclosing Run/Execute call returns nil.
var Terminate error = &Error{Category: ErrCategoryWASM, Code: ErrCodeTerminated, Message: "terminated"}

// IsTrap reports whether err is a WasmEdge execution-phase failure. WasmEdge
// assigns every execution error a code in the 0x04xx range; loader,
// validation, instantiation, limit, interruption, and user-level errors are
// deliberately not classified as traps. Wrapped and joined error trees are
// supported; every branch is checked rather than stopping at the first
// *Error.
func IsTrap(err error) bool {
	if err == nil {
		return false
	}
	if wasmErr, ok := err.(*Error); ok && isTrapError(wasmErr) {
		return true
	}
	// Preserve errors.As compatibility for error types that synthesize an
	// *Error through a custom As method, while traversing Unwrap ourselves so
	// an earlier non-trap branch cannot hide a later trap.
	if as, ok := err.(interface{ As(any) bool }); ok {
		var wasmErr *Error
		if as.As(&wasmErr) && isTrapError(wasmErr) {
			return true
		}
	}
	switch err := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range err.Unwrap() {
			if IsTrap(child) {
				return true
			}
		}
	case interface{ Unwrap() error }:
		return IsTrap(err.Unwrap())
	}
	return false
}

func isTrapError(err *Error) bool {
	return err != nil &&
		err.Category == ErrCategoryWASM &&
		err.Code >= 0x0400 &&
		err.Code <= 0x04FF
}

// IsCostLimitExceeded reports whether err says that a Statistics cost limit
// stopped execution. Wrapped errors are supported.
func IsCostLimitExceeded(err error) bool {
	return errors.Is(err, &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeCostLimitExceeded,
	})
}

// newResult converts a C result into a Go error. Success and graceful
// termination both map to nil, mirroring WasmEdge_ResultOK.
func newResult(res C.WasmEdge_Result) error {
	if C.WasmEdge_ResultOK(res) {
		return nil
	}
	category := ErrCategory(C.WasmEdge_ResultGetCategory(res))
	code := ErrCode(C.WasmEdge_ResultGetCode(res))
	if category == ErrCategoryUserLevel {
		if err := takeHostError(code); err != nil {
			return err
		}
	}
	e := &Error{
		Category: category,
		Code:     code,
	}
	if e.Category == ErrCategoryWASM {
		// WasmEdge 0.17.1's ResultGetMessage indexes its native message table
		// without checking the caller-controlled 24-bit code. Only ask it for
		// codes in the complete 0.17.1 enum; unknown codes use the safe Go
		// fallback and remain debuggable.
		if code.validWASM() {
			e.Message = C.GoString(C.WasmEdge_ResultGetMessage(res))
		} else {
			e.Message = code.String()
		}
	}
	return e
}

// toResult converts a host-function error into the C result the engine
// expects: nil => Success, Terminate => Terminate, and a valid non-success
// WASM *Error => its category/code. Other Go errors receive a private
// user-level token; when the result returns to newResult, the original error
// is restored.
func toResult(err error) C.WasmEdge_Result {
	if err == nil {
		return C.WasmEdge_ResultGen(C.WasmEdge_ErrCategory_WASM, C.uint32_t(ErrCodeSuccess))
	}
	if errors.Is(err, Terminate) {
		return C.WasmEdge_ResultGen(C.WasmEdge_ErrCategory_WASM, C.uint32_t(ErrCodeTerminated))
	}
	var we *Error
	if errors.As(err, &we) && we != nil &&
		we.Category == ErrCategoryWASM &&
		we.Code != ErrCodeSuccess &&
		we.Code.validWASM() {
		return C.WasmEdge_ResultGen(
			C.WasmEdge_ErrCategory_WASM,
			C.uint32_t(we.Code),
		)
	}
	// Preserve user-level and malformed *Error values through the private
	// token registry. Passing an unknown WASM code to the native runtime is
	// unsafe in 0.17.1, and accepting an arbitrary user-level token could
	// consume another callback's registered Go error.
	return C.WasmEdge_ResultGen(
		C.WasmEdge_ErrCategory_UserLevelError,
		C.uint32_t(registerHostError(err)),
	)
}

func (c ErrCode) validWASM() bool {
	switch {
	case c <= ErrCodeAOTNotImpl:
	case c >= ErrCodeIllegalPath && c <= ErrCodeMalformedCatchFlags:
	case c >= ErrCodeMalformedSort && c <= ErrCodeComponentNotImplLoader:
	case c >= ErrCodeInvalidAlignment && c <= ErrCodeInvalidOffset:
	case c >= ErrCodeMissingArgument && c <= ErrCodeInvalidExternName:
	case c >= ErrCodeModuleNameConflict && c <= ErrCodeElemSegDoesNotFit:
	case c >= ErrCodeInvalidCoreSort && c <= ErrCodeComponentNotImplInstantiate:
	case c >= ErrCodeWrongInstanceAddress && c <= ErrCodeUncaughtException:
	default:
		return false
	}
	return true
}

const (
	hostErrorTokenBit  = ErrCode(0x800000)
	hostErrorTokenMask = ErrCode(0x7FFFFF)
)

type hostErrorEntry struct {
	err error
}

var hostErrorRegistry = struct {
	sync.Mutex
	next    ErrCode
	entries map[ErrCode]*hostErrorEntry
}{
	entries: make(map[ErrCode]*hostErrorEntry),
}

func registerHostError(err error) ErrCode {
	hostErrorRegistry.Lock()
	defer hostErrorRegistry.Unlock()
	for {
		hostErrorRegistry.next = (hostErrorRegistry.next + 1) & hostErrorTokenMask
		code := hostErrorTokenBit | hostErrorRegistry.next
		if _, exists := hostErrorRegistry.entries[code]; exists {
			continue
		}
		entry := &hostErrorEntry{err: err}
		hostErrorRegistry.entries[code] = entry
		return code
	}
}

func takeHostError(code ErrCode) error {
	if code&hostErrorTokenBit == 0 {
		return nil
	}
	hostErrorRegistry.Lock()
	entry := hostErrorRegistry.entries[code]
	if entry != nil {
		delete(hostErrorRegistry.entries, code)
	}
	hostErrorRegistry.Unlock()
	if entry == nil {
		return nil
	}
	return entry.err
}

// resultError builds the error for a synthetic result; it exists so tests
// can exercise newResult without cgo in _test.go files.
func resultError(cat ErrCategory, code ErrCode) error {
	return newResult(C.WasmEdge_ResultGen(C.enum_WasmEdge_ErrCategory(cat), C.uint32_t(code)))
}

func roundTripHostError(err error) error {
	return newResult(toResult(err))
}
