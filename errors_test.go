package wasmedge

import (
	"errors"
	"fmt"
	"testing"
)

func TestResultMapping(t *testing.T) {
	if err := resultError(ErrCategoryWASM, ErrCodeSuccess); err != nil {
		t.Fatalf("success mapped to %v, want nil", err)
	}
	// Graceful termination is a successful outcome (mirrors WasmEdge_ResultOK).
	if err := resultError(ErrCategoryWASM, ErrCodeTerminated); err != nil {
		t.Fatalf("terminate mapped to %v, want nil", err)
	}

	err := resultError(ErrCategoryWASM, ErrCodeFuncNotFound)
	var we *Error
	if !errors.As(err, &we) {
		t.Fatalf("want *Error, got %T", err)
	}
	if we.Category != ErrCategoryWASM || we.Code != ErrCodeFuncNotFound {
		t.Fatalf("got %+v", we)
	}
	if we.Message == "" {
		t.Fatal("WASM-category error lost its engine message")
	}
}

func TestErrorIsMatchesByCategoryAndCode(t *testing.T) {
	a := &Error{Category: ErrCategoryWASM, Code: ErrCodeCostLimitExceeded, Message: "x"}
	b := &Error{Category: ErrCategoryWASM, Code: ErrCodeCostLimitExceeded, Message: "different"}
	if !errors.Is(a, b) {
		t.Fatal("same category+code should match regardless of message")
	}
	c := &Error{Category: ErrCategoryUserLevel, Code: ErrCodeCostLimitExceeded}
	if errors.Is(a, c) {
		t.Fatal("different category must not match")
	}
	if !errors.Is(fmt.Errorf("wrapped: %w", a), b) {
		t.Fatal("wrapping must not break matching")
	}
}

func TestErrorString(t *testing.T) {
	e := &Error{Category: ErrCategoryWASM, Code: ErrCodeFuncNotFound, Message: "wasm function not found"}
	got := e.Error()
	want := "wasmedge: wasm function not found (wasm code 0x0005)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// TODO(intern-easy): A1 — after completing the ErrCode table, add
// TestErrCodeString covering at least one code from each phase range
// (load 0x01xx, validation 0x02xx, instantiation 0x03xx, execution 0x04xx+).
