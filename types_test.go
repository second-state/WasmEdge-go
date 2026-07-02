package wasmedge

import "testing"

func TestFunctionTypeRoundTrip(t *testing.T) {
	ft := NewFunctionType(
		[]ValType{ValTypeI32(), ValTypeI64()},
		[]ValType{ValTypeF64()},
	)
	if ft == nil {
		t.Fatal("NewFunctionType returned nil")
	}
	defer ft.Close()

	params, results := ft.Parameters(), ft.Results()
	if len(params) != 2 || !params[0].IsI32() || !params[1].IsI64() {
		t.Fatalf("params: %v", params)
	}
	if len(results) != 1 || !results[0].IsF64() {
		t.Fatalf("results: %v", results)
	}
	if err := ft.Close(); err != nil {
		t.Fatal("double Close must be a no-op")
	}
}

func TestEmptyFunctionType(t *testing.T) {
	ft := NewFunctionType(nil, nil)
	if ft == nil {
		t.Fatal("nil/nil function type must be valid (nullary function)")
	}
	defer ft.Close()
	if len(ft.Parameters()) != 0 || len(ft.Results()) != 0 {
		t.Fatal("expected empty parameter and result lists")
	}
}

func TestTableTypeRoundTrip(t *testing.T) {
	lim := Limits{Min: 2, Max: 10, HasMax: true}
	tt := NewTableType(ValTypeFuncRef(), lim)
	if tt == nil {
		t.Fatal("NewTableType returned nil")
	}
	defer tt.Close()
	if !tt.RefType().IsFuncRef() {
		t.Errorf("ref type: %s", tt.RefType())
	}
	if got := tt.Limits(); got != lim {
		t.Errorf("limits: got %+v want %+v", got, lim)
	}
}

func TestTableTypeRejectsNonRef(t *testing.T) {
	if tt := NewTableType(ValTypeI32(), Limits{Min: 1}); tt != nil {
		t.Fatal("i32 element type must be rejected")
	}
}

func TestMemoryTypeRoundTrip(t *testing.T) {
	lim := Limits{Min: 1, Max: 4, HasMax: true}
	mt := NewMemoryType(lim)
	if mt == nil {
		t.Fatal("NewMemoryType returned nil")
	}
	defer mt.Close()
	if got := mt.Limits(); got != lim {
		t.Errorf("limits: got %+v want %+v", got, lim)
	}
}

func TestMemory64TypeRoundTrip(t *testing.T) {
	lim := Limits{Min: 1, Max: 8, HasMax: true, Is64: true}
	mt := NewMemoryType(lim)
	if mt == nil {
		t.Skip("Memory64 not enabled in this library build")
	}
	defer mt.Close()
	if got := mt.Limits(); !got.Is64 {
		t.Errorf("lost Is64: %+v", got)
	}
}

func TestGlobalTypeRoundTrip(t *testing.T) {
	gt := NewGlobalType(ValTypeI64(), MutabilityVar)
	if gt == nil {
		t.Fatal("NewGlobalType returned nil")
	}
	defer gt.Close()
	if !gt.ValType().IsI64() || gt.Mutability() != MutabilityVar {
		t.Fatalf("got %s/%s", gt.ValType(), gt.Mutability())
	}
}

// TODO(intern-easy): A3 — when the ImportType/ExportType accessors land in
// astmodule.go, add TestImportExportTypes here loading
// internal/testwasm.HostCallModule and asserting module/name/external-type
// of each entry. Pattern: TestFunctionTypeRoundTrip above.
