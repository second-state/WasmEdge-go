package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"errors"
	"fmt"
)

// ErrCategory identifies who produced an error code: the WASM runtime or the
// embedder (host functions returning custom errors).
type ErrCategory uint32

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

// Runtime-phase error codes. This set is complete; the loading, validation,
// instantiation and execution phase codes (0x01xx-0x06xx ranges) are tracked
// as intern task A1.
//
// TODO(intern-easy): A1 — complete the ErrCode constants and the String()
// table from the `UseErrCode` section of WasmEdge's include/common/enum.inc
// (0.17.x). Keep the Go names identical to the upstream enum names (e.g.
// E(MalformedMagic, 0x0103, ...) => ErrCodeMalformedMagic). Extend
// TestErrCodeString in errors_test.go with at least one code per phase.
const (
	ErrCodeSuccess              ErrCode = 0x0000
	ErrCodeTerminated           ErrCode = 0x0001
	ErrCodeRuntimeError         ErrCode = 0x0002
	ErrCodeCostLimitExceeded    ErrCode = 0x0003
	ErrCodeWrongVMWorkflow      ErrCode = 0x0004
	ErrCodeFuncNotFound         ErrCode = 0x0005
	ErrCodeAOTDisabled          ErrCode = 0x0006
	ErrCodeInterrupted          ErrCode = 0x0007
	ErrCodeNotValidated         ErrCode = 0x0008
	ErrCodeNonNullRequired      ErrCode = 0x0009
	ErrCodeSetValueToConst      ErrCode = 0x000A
	ErrCodeSetValueErrorType    ErrCode = 0x000B
	ErrCodeUserDefError         ErrCode = 0x000C
	ErrCodeInvalidAOTConfigure  ErrCode = 0x000D
	ErrCodeAOTNotImpl           ErrCode = 0x000E
	ErrCodeLazyCompilationError ErrCode = 0x000F
)

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

// newResult converts a C result into a Go error. Success and graceful
// termination both map to nil, mirroring WasmEdge_ResultOK.
func newResult(res C.WasmEdge_Result) error {
	if C.WasmEdge_ResultOK(res) {
		return nil
	}
	e := &Error{
		Category: ErrCategory(C.WasmEdge_ResultGetCategory(res)),
		Code:     ErrCode(C.WasmEdge_ResultGetCode(res)),
	}
	if e.Category == ErrCategoryWASM {
		e.Message = C.GoString(C.WasmEdge_ResultGetMessage(res))
	}
	return e
}

// toResult converts a host-function error into the C result the engine
// expects: nil => Success, Terminate => Terminate, *Error => its
// category/code, anything else => a user-level failure carrying the
// UserDefError code (the engine keeps no message for user-level results;
// the Go error text is intentionally not smuggled across the boundary).
func toResult(err error) C.WasmEdge_Result {
	if err == nil {
		return C.WasmEdge_ResultGen(C.WasmEdge_ErrCategory_WASM, C.uint32_t(ErrCodeSuccess))
	}
	if errors.Is(err, Terminate) {
		return C.WasmEdge_ResultGen(C.WasmEdge_ErrCategory_WASM, C.uint32_t(ErrCodeTerminated))
	}
	var we *Error
	if errors.As(err, &we) {
		return C.WasmEdge_ResultGen(C.enum_WasmEdge_ErrCategory(we.Category), C.uint32_t(we.Code))
	}
	return C.WasmEdge_ResultGen(C.WasmEdge_ErrCategory_UserLevelError, C.uint32_t(ErrCodeUserDefError))
}

// resultError builds the error for a synthetic result; it exists so tests
// can exercise newResult without cgo in _test.go files.
func resultError(cat ErrCategory, code ErrCode) error {
	return newResult(C.WasmEdge_ResultGen(C.enum_WasmEdge_ErrCategory(cat), C.uint32_t(code)))
}
