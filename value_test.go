package wasmedge

import (
	"math"
	"testing"
)

func TestValueRoundTrips(t *testing.T) {
	if got := I32(-42).I32(); got != -42 {
		t.Errorf("i32: got %d", got)
	}
	if got := I64(math.MinInt64).I64(); got != math.MinInt64 {
		t.Errorf("i64: got %d", got)
	}
	if got := F32(1.5).F32(); got != 1.5 {
		t.Errorf("f32: got %g", got)
	}
	if got := F64(math.Pi).F64(); got != math.Pi {
		t.Errorf("f64: got %g", got)
	}
	lanes := [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if got := V128(lanes).V128(); got != lanes {
		t.Errorf("v128: got %x", got)
	}
}

func TestValueKinds(t *testing.T) {
	cases := []struct {
		v    Value
		kind ValKind
	}{
		{I32(0), ValKindI32},
		{I64(0), ValKindI64},
		{F32(0), ValKindF32},
		{F64(0), ValKindF64},
		{V128([16]byte{}), ValKindV128},
		{NullExternRef(), ValKindExternRef},
		{NullFuncRef(), ValKindFuncRef},
	}
	for _, c := range cases {
		if got := c.v.Kind(); got != c.kind {
			t.Errorf("kind: got %s, want %s", got, c.kind)
		}
	}
}

func TestValueKindMismatchPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("I64 accessor on i32 value must panic")
		}
	}()
	_ = I32(1).I64()
}

func TestNullRefs(t *testing.T) {
	if !NullExternRef().IsNullRef() {
		t.Error("NullExternRef not null")
	}
	if !NullFuncRef().IsNullRef() {
		t.Error("NullFuncRef not null")
	}
	if r := NullExternRef().ExternRef(); r != nil {
		t.Errorf("null externref payload: got %v", r)
	}
}

func TestExternRefPinning(t *testing.T) {
	type payload struct{ n int }
	p := &payload{n: 7}

	ref := NewExternRef(p)
	val := ExternRefValue(ref)
	if val.Kind() != ValKindExternRef || val.IsNullRef() {
		t.Fatalf("unexpected value %v", val)
	}

	back := val.ExternRef()
	if back == nil {
		t.Fatal("lost the payload")
	}
	if got := back.Value().(*payload); got != p {
		t.Fatalf("payload identity lost: got %p want %p", got, p)
	}
	// Borrowed views must not release the pin.
	if err := back.Close(); err != nil {
		t.Fatal(err)
	}
	if got := ref.Value().(*payload); got != p {
		t.Fatal("borrowed Close released the pin")
	}

	// Owner Close releases; double Close is a no-op.
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExternRefUseAfterClosePanics(t *testing.T) {
	ref := NewExternRef("x")
	_ = ref.Close()
	defer func() {
		if recover() == nil {
			t.Fatal("Value after Close must panic")
		}
	}()
	_ = ref.Value()
}
