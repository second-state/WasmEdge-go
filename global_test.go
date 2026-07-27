package wasmedge

import (
	"errors"
	"testing"
)

func TestGlobalValueAndSetValue(t *testing.T) {
	g := newTestGlobal(t, GlobalType{
		Value: ValTypeI64(), Mutability: MutabilityVar,
	}, I64(9))
	defer g.Close()

	if got := g.Value().I64(); got != 9 {
		t.Fatalf("initial value: got %d, want 9", got)
	}
	if err := g.SetValue(I64(-27)); err != nil {
		t.Fatal(err)
	}
	if got := g.Value().I64(); got != -27 {
		t.Fatalf("updated value: got %d, want -27", got)
	}
}

func TestGlobalV128RoundTrip(t *testing.T) {
	initial := [16]byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff,
	}
	g := newTestGlobal(t, GlobalType{
		Value: ValTypeV128(), Mutability: MutabilityVar,
	}, V128(initial))
	defer g.Close()
	if got := g.Value().V128(); got != initial {
		t.Fatalf("initial v128: got %x, want %x", got, initial)
	}

	updated := [16]byte{
		0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88,
		0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00,
	}
	if err := g.SetValue(V128(updated)); err != nil {
		t.Fatal(err)
	}
	if got := g.Value().V128(); got != updated {
		t.Fatalf("updated v128: got %x, want %x", got, updated)
	}
}

func TestGlobalSetValueErrors(t *testing.T) {
	constGlobal := newTestGlobal(t, GlobalType{
		Value: ValTypeI64(), Mutability: MutabilityConst,
	}, I64(1))
	defer constGlobal.Close()

	err := constGlobal.SetValue(I64(2))
	assertGlobalErrorCode(t, err, ErrCodeSetValueToConst)
	if got := constGlobal.Value().I64(); got != 1 {
		t.Fatalf("const global changed after rejected SetValue: got %d", got)
	}

	varGlobal := newTestGlobal(t, GlobalType{
		Value: ValTypeI64(), Mutability: MutabilityVar,
	}, I64(3))
	defer varGlobal.Close()

	err = varGlobal.SetValue(I32(4))
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("mismatched SetValue: got %v, want ErrInvalidArgument", err)
	}
	if got := varGlobal.Value().I64(); got != 3 {
		t.Fatalf("global changed after type mismatch: got %d", got)
	}
}

func TestGlobalRetainsReferenceOwner(t *testing.T) {
	first := NewExternRef("first")
	g := newTestGlobal(t, GlobalType{
		Value: ValTypeExternRef(), Mutability: MutabilityVar,
	}, ExternRefValue(first))
	if err := first.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("global did not lease its initial reference owner: %v", err)
	}

	previous := g.Value()
	if got := previous.ExternRef().Value(); got != "first" {
		t.Fatalf("initial externref: got %v", got)
	}

	second := NewExternRef("second")
	if err := g.SetValue(ExternRefValue(second)); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("global did not lease its updated reference owner: %v", err)
	}
	if got := g.Value().ExternRef().Value(); got != "second" {
		t.Fatalf("updated externref: got %v", got)
	}
	// A Value is a copy, so changing the global must not invalidate a
	// reference Value that was already returned.
	if got := previous.ExternRef().Value(); got != "first" {
		t.Fatalf("previous externref after update: got %v", got)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGlobalRejectsClosedValueOwner(t *testing.T) {
	g := newTestGlobal(t, GlobalType{
		Value: ValTypeExternRef(), Mutability: MutabilityVar,
	}, NullExternRef())
	defer g.Close()

	ref := NewExternRef("closed")
	v := ExternRefValue(ref)
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}
	assertGlobalPanic(t, func() {
		_ = g.SetValue(v)
	})
	if !g.Value().IsNullRef() {
		t.Fatal("global changed after a Value with a closed owner was rejected")
	}
}

func TestGlobalFailedSetDoesNotRetainReferenceOwner(t *testing.T) {
	g := newTestGlobal(t, GlobalType{
		Value: ValTypeExternRef(), Mutability: MutabilityConst,
	}, NullExternRef())
	defer g.Close()

	ref := NewExternRef("rejected")
	err := g.SetValue(ExternRefValue(ref))
	assertGlobalErrorCode(t, err, ErrCodeSetValueToConst)
	if rooted := g.roots.owner(0, ExternRefValue(ref)); rooted != nil {
		t.Fatalf("failed SetValue retained reference root %#v", rooted)
	}
	if err := ref.Close(); err != nil {
		t.Fatalf("failed SetValue leaked owner lease: %v", err)
	}
}

func TestGlobalUseAfterClosePanics(t *testing.T) {
	g := newTestGlobal(t, GlobalType{
		Value: ValTypeI32(), Mutability: MutabilityVar,
	}, I32(1))
	if err := g.Close(); err != nil {
		t.Fatal(err)
	}

	assertGlobalPanic(t, func() { _ = g.Value() })
	assertGlobalPanic(t, func() { _ = g.SetValue(I32(2)) })
}

func assertGlobalErrorCode(t *testing.T, err error, code ErrCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("SetValue succeeded, want error code %v", code)
	}
	var got *Error
	if !errors.As(err, &got) {
		t.Fatalf("SetValue error type: got %T (%v), want *Error", err, err)
	}
	if got.Category != ErrCategoryWASM || got.Code != code {
		t.Fatalf(
			"SetValue error: got category %v code 0x%04x, want wasm code 0x%04x",
			got.Category,
			uint32(got.Code),
			uint32(code),
		)
	}
	if !errors.Is(err, &Error{Category: ErrCategoryWASM, Code: code}) {
		t.Fatalf("errors.Is did not match wasm code 0x%04x", uint32(code))
	}
}

func newTestGlobal(t *testing.T, typ GlobalType, initial Value) *Global {
	t.Helper()
	global, err := NewGlobal(typ, initial)
	if err != nil {
		t.Fatal(err)
	}
	return global
}

func assertGlobalPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("operation did not panic")
		}
	}()
	fn()
}
