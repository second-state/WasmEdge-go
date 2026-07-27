package wasmedge

import (
	"errors"
	"strings"
	"testing"
)

func TestNewTableRejectsNonNullableASTElement(t *testing.T) {
	// (module (table (export "table") 1 (ref func)))
	//
	// A non-nullable reference type has no meaningful zero value. The
	// no-initializer native constructor therefore cannot create this table;
	// callers must supply a compatible reference through NewTableWithInit.
	wasm := []byte{
		0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00,
		0x04, 0x05, 0x01, 0x64, 0x70, 0x00, 0x01,
		0x07, 0x09, 0x01, 0x05, 't', 'a', 'b', 'l', 'e', 0x01, 0x00,
	}
	loader, err := NewLoader(&Config{EnableProposals: []Proposal{
		ProposalReferenceTypes,
		ProposalFunctionReferences,
		ProposalGC,
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()
	ast, err := loader.LoadBytes(wasm)
	if err != nil {
		t.Fatalf("load non-nullable table fixture: %v", err)
	}
	defer ast.Close()

	exports := ast.Exports()
	if len(exports) != 1 {
		t.Fatalf("exports: got %d, want 1", len(exports))
	}
	tt, ok := exports[0].TableType()
	if !ok {
		t.Fatal("table export did not expose a TableType")
	}
	if !tt.Element.IsRef() || tt.Element.IsRefNull() {
		t.Fatalf("element type: got %s (ref=%v null=%v), want non-nullable reference",
			tt.Element, tt.Element.IsRef(), tt.Element.IsRefNull())
	}

	_, err = NewTable(tt)
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("NewTable: got %v, want ErrInvalidArgument", err)
	}
	if !strings.Contains(err.Error(), "NewTableWithInit") {
		t.Fatalf("NewTable error %q does not guide callers to NewTableWithInit", err)
	}
}

func TestTypedReferenceContainersRequireExactTypes(t *testing.T) {
	nullable0, nullable1, _ := typedFunctionReferenceTypes(t)

	first, err := NewFunction(
		FunctionType{},
		func(*CallContext, []Value) ([]Value, error) { return nil, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFunction(
		FunctionType{Params: []ValType{ValTypeI32()}},
		func(*CallContext, []Value) ([]Value, error) { return nil, nil },
	)
	if err != nil {
		_ = first.Close()
		t.Fatal(err)
	}

	firstValue := valueWithType(FuncRefValue(first), nullable0)
	secondValue := valueWithType(FuncRefValue(second), nullable1)

	var table *Table
	var global *Global
	t.Cleanup(func() {
		if table != nil {
			_ = table.Close()
		}
		if global != nil {
			_ = global.Close()
		}
		_ = first.Close()
		_ = second.Close()
	})

	wrongTable, err := NewTableWithInit(TableType{
		Element: nullable0,
		Limits:  Limits{Min: 1},
	}, secondValue)
	if wrongTable != nil {
		_ = wrongTable.Close()
	}
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("mismatched typed table initializer: got %v, want ErrInvalidArgument", err)
	}

	table, err = NewTableWithInit(TableType{
		Element: nullable0,
		Limits:  Limits{Min: 1},
	}, firstValue)
	if err != nil {
		t.Fatal(err)
	}
	if err := table.Set(0, secondValue); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("mismatched typed table Set: got %v, want ErrInvalidArgument", err)
	}
	if got, err := table.Get(0); err != nil {
		t.Fatal(err)
	} else if got.FuncRef().ptr != first.ptr {
		t.Fatal("failed typed table Set changed the stored function")
	}

	wrongGlobal, err := NewGlobal(GlobalType{
		Value:      nullable0,
		Mutability: MutabilityVar,
	}, secondValue)
	if wrongGlobal != nil {
		_ = wrongGlobal.Close()
	}
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("mismatched typed global initializer: got %v, want ErrInvalidArgument", err)
	}

	global, err = NewGlobal(GlobalType{
		Value:      nullable0,
		Mutability: MutabilityVar,
	}, firstValue)
	if err != nil {
		t.Fatal(err)
	}
	if err := global.SetValue(secondValue); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("mismatched typed global SetValue: got %v, want ErrInvalidArgument", err)
	}
	if got := global.Value().FuncRef(); got.ptr != first.ptr {
		t.Fatal("failed typed global SetValue changed the stored function")
	}

	if err := second.Close(); err != nil {
		t.Fatalf("rejected typed reference retained its owner: %v", err)
	}
}

func TestTypedReferenceContainersEnforceNullability(t *testing.T) {
	nullable, _, nonNullable := typedFunctionReferenceTypes(t)

	nullableNull := valueWithType(NullFuncRef(), nullable)
	table, err := NewTableWithInit(TableType{
		Element: nullable,
		Limits:  Limits{Min: 1},
	}, nullableNull)
	if err != nil {
		t.Fatalf("nullable typed table rejected null: %v", err)
	}
	defer table.Close()

	global, err := NewGlobal(GlobalType{
		Value:      nullable,
		Mutability: MutabilityVar,
	}, nullableNull)
	if err != nil {
		t.Fatalf("nullable typed global rejected null: %v", err)
	}
	defer global.Close()

	nonNullableNull := valueWithType(NullFuncRef(), nonNullable)
	if got, err := NewTableWithInit(TableType{
		Element: nonNullable,
		Limits:  Limits{Min: 1},
	}, nonNullableNull); !errors.Is(err, ErrInvalidArgument) {
		if got != nil {
			_ = got.Close()
		}
		t.Fatalf("non-nullable typed table initializer: got %v, want ErrInvalidArgument", err)
	}
	if got, err := NewGlobal(GlobalType{
		Value:      nonNullable,
		Mutability: MutabilityVar,
	}, nonNullableNull); !errors.Is(err, ErrInvalidArgument) {
		if got != nil {
			_ = got.Close()
		}
		t.Fatalf("non-nullable typed global initializer: got %v, want ErrInvalidArgument", err)
	}

	function, err := NewFunction(
		FunctionType{},
		func(*CallContext, []Value) ([]Value, error) { return nil, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	defer function.Close()
	nonNullableValue := valueWithType(FuncRefValue(function), nonNullable)

	nonNullableTable, err := NewTableWithInit(TableType{
		Element: nonNullable,
		Limits:  Limits{Min: 1},
	}, nonNullableValue)
	if err != nil {
		t.Fatal(err)
	}
	defer nonNullableTable.Close()
	if err := nonNullableTable.Set(0, nonNullableNull); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("non-nullable typed table Set: got %v, want ErrInvalidArgument", err)
	}

	nonNullableGlobal, err := NewGlobal(GlobalType{
		Value:      nonNullable,
		Mutability: MutabilityVar,
	}, nonNullableValue)
	if err != nil {
		t.Fatal(err)
	}
	defer nonNullableGlobal.Close()
	if err := nonNullableGlobal.SetValue(nonNullableNull); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("non-nullable typed global SetValue: got %v, want ErrInvalidArgument", err)
	}
}

func typedFunctionReferenceTypes(t *testing.T) (nullable0, nullable1, nonNullable ValType) {
	t.Helper()

	// (module
	//   (type (func))
	//   (type (func (param i32)))
	//   (table (export "a") 1 (ref null 0))
	//   (table (export "b") 1 (ref null 1))
	//   (table (export "n") 1 (ref 0)))
	wasm := []byte{
		0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00,
		0x01, 0x08, 0x02,
		0x60, 0x00, 0x00,
		0x60, 0x01, 0x7F, 0x00,
		0x04, 0x0D, 0x03,
		0x63, 0x00, 0x00, 0x01,
		0x63, 0x01, 0x00, 0x01,
		0x64, 0x00, 0x00, 0x01,
		0x07, 0x0D, 0x03,
		0x01, 'a', 0x01, 0x00,
		0x01, 'b', 0x01, 0x01,
		0x01, 'n', 0x01, 0x02,
	}
	loader, err := NewLoader(&Config{EnableProposals: []Proposal{
		ProposalReferenceTypes,
		ProposalFunctionReferences,
		ProposalGC,
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()
	ast, err := loader.LoadBytes(wasm)
	if err != nil {
		t.Fatalf("load typed-reference fixture: %v", err)
	}
	defer ast.Close()

	types := make(map[string]ValType, 3)
	for _, export := range ast.Exports() {
		tableType, ok := export.TableType()
		if !ok {
			t.Fatalf("export %q is not a table", export.Name())
		}
		types[export.Name()] = tableType.Element
	}
	nullable0, ok0 := types["a"]
	nullable1, ok1 := types["b"]
	nonNullable, okN := types["n"]
	if !ok0 || !ok1 || !okN {
		t.Fatalf("typed-reference fixture exports: got %v", types)
	}
	if nullable0.Equal(nullable1) {
		t.Fatal("fixture type indices unexpectedly compare equal")
	}
	if !nullable0.IsFuncRef() || !nullable1.IsFuncRef() ||
		!nullable0.IsRefNull() || !nullable1.IsRefNull() {
		t.Fatal("fixture nullable function reference types are invalid")
	}
	if !nonNullable.IsFuncRef() || nonNullable.IsRefNull() {
		t.Fatal("fixture non-nullable function reference type is invalid")
	}
	return nullable0, nullable1, nonNullable
}

func valueWithType(value Value, typ ValType) Value {
	value.raw.Type = typ.raw
	return value
}

func TestTableTypeSizeGrowAndFuncRef(t *testing.T) {
	tt := TableType{
		Element: ValTypeFuncRef(),
		Limits:  Limits{Min: 2, Max: 3, HasMax: true},
	}

	fn, err := NewFunction(FunctionType{}, func(*CallContext, []Value) ([]Value, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fn.Close()

	tbl, err := NewTable(tt)
	if err != nil {
		t.Fatal(err)
	}
	defer tbl.Close()

	gotType := tbl.Type()
	if !gotType.Element.IsFuncRef() {
		t.Fatalf("element type: got %s, want funcref", gotType.Element)
	}
	if got := gotType.Limits; got.Min != 2 || got.Max != 3 || !got.HasMax {
		t.Fatalf("limits: got %+v, want {Min:2 Max:3 HasMax:true}", got)
	}
	if got := tbl.Size(); got != 2 {
		t.Fatalf("initial size: got %d, want 2", got)
	}

	if err := tbl.Set(1, FuncRefValue(fn)); err != nil {
		t.Fatal(err)
	}
	got, err := tbl.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind() != ValKindFuncRef || got.IsNullRef() {
		t.Fatalf("Get returned %v, want non-null funcref", got)
	}
	if got.FuncRef() == nil {
		t.Fatal("FuncRef returned nil")
	}

	if err := tbl.Grow(1); err != nil {
		t.Fatal(err)
	}
	if got := tbl.Size(); got != 3 {
		t.Fatalf("grown size: got %d, want 3", got)
	}
	grown, err := tbl.Get(2)
	if err != nil {
		t.Fatal(err)
	}
	if !grown.IsNullRef() {
		t.Fatalf("new slot: got %v, want null funcref", grown)
	}

	var we *Error
	if err := tbl.Grow(1); !errors.As(err, &we) {
		t.Fatalf("growth beyond maximum must return *Error, got %v", err)
	}
	if got := tbl.Size(); got != 3 {
		t.Fatalf("failed Grow changed size to %d", got)
	}
	if _, err := tbl.Get(99); !errors.As(err, &we) {
		t.Fatalf("out-of-bounds Get must return *Error, got %v", err)
	}
}

func TestTable64HostSizeChecks(t *testing.T) {
	const maxUint64 = ^uint64(0)

	tt := TableType{
		Element: ValTypeFuncRef(),
		Limits: Limits{
			Max:    maxUint64,
			HasMax: true,
			Is64:   true,
		},
	}
	tbl, err := NewTable(tt)
	if err != nil {
		t.Fatalf("zero-sized table64 with uint64 maximum: %v", err)
	}
	defer tbl.Close()

	got := tbl.Type().Limits
	if !got.Is64 || !got.HasMax || got.Max != maxUint64 {
		t.Fatalf("table64 descriptor changed: got %+v", got)
	}
	if err := tbl.Grow(maxUint64); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Grow(MaxUint64): got %v, want ErrUnavailable", err)
	}
	if got := tbl.Size(); got != 0 {
		t.Fatalf("failed Grow changed table size to %d", got)
	}

	nonempty, err := NewTable(TableType{
		Element: ValTypeFuncRef(),
		Limits:  Limits{Min: 1, Is64: true},
	})
	if err != nil {
		t.Fatalf("one-element table64: %v", err)
	}
	defer nonempty.Close()
	for name, delta := range map[string]uint64{
		"sum overflow":        maxUint64,
		"host index overflow": uint64(maxInt()),
	} {
		if err := nonempty.Grow(delta); !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Grow %s: got %v, want ErrUnavailable", name, err)
		}
		if got := nonempty.Size(); got != 1 {
			t.Fatalf("failed Grow %s changed table size to %d", name, got)
		}
	}

	unrepresentable := TableType{
		Element: ValTypeFuncRef(),
		Limits:  Limits{Min: maxUint64, Is64: true},
	}
	if _, err := NewTable(unrepresentable); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("NewTable(MaxUint64): got %v, want ErrUnavailable", err)
	}
	if _, err := NewTableWithInit(unrepresentable, NullFuncRef()); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("NewTableWithInit(MaxUint64): got %v, want ErrUnavailable", err)
	}
}

func TestCopiedTableTypeOutlivesTable(t *testing.T) {
	tbl, err := NewTable(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	descriptor := tbl.Type()
	if err := tbl.Close(); err != nil {
		t.Fatal(err)
	}
	if descriptor.Limits.Min != 1 || !descriptor.Element.IsExternRef() {
		t.Fatalf("descriptor changed after table close: %+v", descriptor)
	}
}

func TestNewTableWithInitRetainsReference(t *testing.T) {
	ref := NewExternRef("initial")
	table, err := NewTableWithInit(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 2},
	}, ExternRefValue(ref))
	if err != nil {
		t.Fatal(err)
	}
	if err := ref.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("initializer owner was not leased: %v", err)
	}
	for i := uint64(0); i < 2; i++ {
		value, err := table.Get(i)
		if err != nil {
			t.Fatal(err)
		}
		if got := value.ExternRef().Value(); got != "initial" {
			t.Fatalf("slot %d = %v, want initial", i, got)
		}
	}
	if err := table.Close(); err != nil {
		t.Fatal(err)
	}
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTableRejectsClosedReferenceOwner(t *testing.T) {
	tbl, err := NewTable(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tbl.Close()

	ref := NewExternRef("closed")
	value := ExternRefValue(ref)
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() == nil {
			t.Fatal("Set accepted a Value whose ExternRef owner was closed")
		}
	}()
	_ = tbl.Set(0, value)
}

func TestTableFailedSetDoesNotRetainReferenceOwner(t *testing.T) {
	tbl, err := NewTable(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer tbl.Close()

	ref := NewExternRef("out-of-bounds")
	err = tbl.Set(99, ExternRefValue(ref))
	var we *Error
	if !errors.As(err, &we) || we.Code != ErrCodeTableOutOfBounds {
		t.Fatalf("out-of-bounds Set: got %v, want ErrCodeTableOutOfBounds", err)
	}
	if rooted := tbl.roots.owner(99, ExternRefValue(ref)); rooted != nil {
		t.Fatalf("failed Set retained reference root %#v", rooted)
	}
	if err := ref.Close(); err != nil {
		t.Fatalf("failed Set leaked owner lease: %v", err)
	}
}

func TestTableSuccessfulOverwriteReleasesPreviousReferenceOwner(t *testing.T) {
	tbl, err := NewTable(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 1},
	})
	if err != nil {
		t.Fatal(err)
	}

	first := NewExternRef("first")
	second := NewExternRef("second")
	t.Cleanup(func() {
		_ = tbl.Close()
		_ = first.Close()
		_ = second.Close()
	})

	if err := tbl.Set(0, ExternRefValue(first)); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("stored owner was not leased: got %v, want ErrInUse", err)
	}

	if err := tbl.Set(0, ExternRefValue(second)); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("overwrite retained previous owner: %v", err)
	}
	if err := second.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("replacement owner was not leased: got %v, want ErrInUse", err)
	}

	if err := tbl.Close(); err != nil {
		t.Fatal(err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("table Close did not release replacement owner: %v", err)
	}
}
