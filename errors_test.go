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

	// Unknown native codes must stay debuggable without calling
	// WasmEdge_ResultGetMessage, whose 0.17.1 implementation does not bounds
	// check its message-table index.
	err = resultError(ErrCategoryWASM, ErrCode(0x00FFFF))
	if !errors.As(err, &we) {
		t.Fatalf("unknown code: want *Error, got %T", err)
	}
	if we.Code != ErrCode(0x00FFFF) || we.Message != "error code 0xffff" {
		t.Fatalf("unknown code: got %+v", we)
	}
}

func TestHostErrorRejectsUnsafeNativeResultValues(t *testing.T) {
	tests := []*Error{
		{
			Category: ErrCategoryWASM,
			Code:     ErrCode(0x00FFFF),
			Message:  "unknown WASM code",
		},
		{
			Category: ErrCategoryUserLevel,
			Code:     hostErrorTokenBit | 1,
			Message:  "must not collide with the private registry",
		},
		{
			Category: ErrCategory(255),
			Code:     ErrCodeRuntimeError,
			Message:  "unknown category",
		},
	}
	for _, want := range tests {
		if got := roundTripHostError(want); got != want {
			t.Fatalf("round trip of %#v: got %#v", want, got)
		}
	}

	valid := &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeMemoryOutOfBounds,
		Message:  "caller text is replaced by the canonical runtime text",
	}
	got := roundTripHostError(valid)
	var native *Error
	if !errors.As(got, &native) || native.Code != valid.Code {
		t.Fatalf("valid WASM error: got %#v", got)
	}
	if native.Message != valid.Code.String() {
		t.Fatalf("valid WASM message: got %q, want %q", native.Message, valid.Code.String())
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

func TestErrorPredicates(t *testing.T) {
	trap := fmt.Errorf("invoke: %w", &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeMemoryOutOfBounds,
	})
	if !IsTrap(trap) {
		t.Fatal("wrapped execution error should be a trap")
	}
	for name, err := range map[string]error{
		"nil":           nil,
		"instantiate":   &Error{Category: ErrCategoryWASM, Code: ErrCodeUnknownImport},
		"limit":         &Error{Category: ErrCategoryWASM, Code: ErrCodeCostLimitExceeded},
		"user category": &Error{Category: ErrCategoryUserLevel, Code: ErrCodeMemoryOutOfBounds},
	} {
		if IsTrap(err) {
			t.Errorf("IsTrap(%s) = true, want false", name)
		}
	}

	limit := fmt.Errorf("execute: %w", &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeCostLimitExceeded,
	})
	if !IsCostLimitExceeded(limit) {
		t.Fatal("wrapped cost-limit error should match")
	}
	if IsCostLimitExceeded(&Error{
		Category: ErrCategoryUserLevel,
		Code:     ErrCodeCostLimitExceeded,
	}) {
		t.Fatal("user-level code must not match the WasmEdge cost-limit error")
	}
	if IsCostLimitExceeded(nil) {
		t.Fatal("nil must not match the cost-limit error")
	}
}

func TestIsTrapTraversesErrorTree(t *testing.T) {
	nonTrap := &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeUnknownImport,
	}
	trap := &Error{
		Category: ErrCategoryWASM,
		Code:     ErrCodeMemoryOutOfBounds,
	}
	tree := fmt.Errorf("execute: %w", errors.Join(
		fmt.Errorf("registration: %w", nonTrap),
		errors.Join(
			errors.New("unrelated failure"),
			fmt.Errorf("guest: %w", trap),
		),
	))
	if !IsTrap(tree) {
		t.Fatal("trap in a later errors.Join branch was not detected")
	}

	if IsTrap(errors.Join(
		nonTrap,
		&Error{
			Category: ErrCategoryUserLevel,
			Code:     ErrCodeMemoryOutOfBounds,
		},
	)) {
		t.Fatal("error tree without a WASM execution error was classified as a trap")
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

func TestErrCodeString(t *testing.T) {
	tests := []struct {
		name string
		code ErrCode
		want string
	}{
		{name: "runtime", code: ErrCodeCostLimitExceeded, want: "cost limit exceeded"},
		{name: "load", code: ErrCodeMalformedMagic, want: "magic header not detected"},
		{name: "component load", code: ErrCodeComponentNotImplLoader, want: "component model (loader) not implemented"},
		{name: "validation", code: ErrCodeInvalidAlignment, want: "atomic alignment must be natural"},
		{name: "component validation", code: ErrCodeComponentNotImplValidator, want: "component model (validator) not implemented"},
		{name: "instantiation", code: ErrCodeUnknownImport, want: "unknown import"},
		{name: "component instantiation", code: ErrCodeComponentNotImplInstantiate, want: "component model (instantiation) not implemented"},
		{name: "execution", code: ErrCodeDivideByZero, want: "integer divide by zero"},
		{name: "unknown", code: ErrCode(0x654321), want: "error code 0x654321"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.String(); got != tt.want {
				t.Fatalf("ErrCode(0x%x).String() = %q, want %q", uint32(tt.code), got, tt.want)
			}
		})
	}
}
