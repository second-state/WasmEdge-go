package wasmedge

import (
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestFunctionTypeRoundTrip(t *testing.T) {
	want := FunctionType{
		Params:  []ValType{ValTypeI32(), ValTypeI64()},
		Results: []ValType{ValTypeF64()},
	}
	fn, err := NewFunction(want, func(*CallContext, []Value) ([]Value, error) {
		return []Value{F64(0)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fn.Close()

	got := fn.Type()
	if len(got.Params) != 2 || !got.Params[0].IsI32() || !got.Params[1].IsI64() {
		t.Fatalf("params: %v", got.Params)
	}
	if len(got.Results) != 1 || !got.Results[0].IsF64() {
		t.Fatalf("results: %v", got.Results)
	}
}

func TestEmptyFunctionType(t *testing.T) {
	fn, err := NewFunction(FunctionType{}, func(*CallContext, []Value) ([]Value, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer fn.Close()
	if ft := fn.Type(); len(ft.Params) != 0 || len(ft.Results) != 0 {
		t.Fatal("expected empty parameter and result lists")
	}
}

func TestTableTypeRoundTrip(t *testing.T) {
	lim := Limits{Min: 2, Max: 10, HasMax: true}
	table, err := NewTable(TableType{Element: ValTypeFuncRef(), Limits: lim})
	if err != nil {
		t.Fatal(err)
	}
	defer table.Close()
	tt := table.Type()
	if !tt.Element.IsFuncRef() {
		t.Errorf("element type: %s", tt.Element)
	}
	if got := tt.Limits; got != lim {
		t.Errorf("limits: got %+v want %+v", got, lim)
	}
}

func TestTableTypeRejectsNonRef(t *testing.T) {
	if _, err := NewTable(TableType{
		Element: ValTypeI32(),
		Limits:  Limits{Min: 1},
	}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("i32 element type: got %v, want ErrInvalidArgument", err)
	}
}

func TestMemoryTypeRoundTrip(t *testing.T) {
	lim := Limits{Min: 1, Max: 4, HasMax: true}
	memory, err := NewMemory(MemoryType{Limits: lim})
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()
	if got := memory.Type().Limits; got != lim {
		t.Errorf("limits: got %+v want %+v", got, lim)
	}
}

func TestLimitsEqual(t *testing.T) {
	if !(Limits{Min: 1}).Equal(Limits{Min: 1, Max: 99}) {
		t.Fatal("Max must be ignored when both limits omit it")
	}
	if (Limits{Min: 1}).Equal(Limits{Min: 1, Max: 2, HasMax: true}) {
		t.Fatal("limits with and without a maximum must differ")
	}
	if (Limits{Min: 1, Is64: true}).Equal(Limits{Min: 1}) {
		t.Fatal("address width must participate in equality")
	}
}

func TestMemory64TypeRoundTrip(t *testing.T) {
	lim := Limits{Min: 1, Max: 8, HasMax: true, Is64: true}
	memory, err := NewMemory(MemoryType{Limits: lim})
	if err != nil {
		t.Skip("Memory64 not enabled in this library build")
	}
	defer memory.Close()
	if got := memory.Type().Limits; !got.Is64 {
		t.Errorf("lost Is64: %+v", got)
	}
}

func TestGlobalTypeRoundTrip(t *testing.T) {
	global, err := NewGlobal(GlobalType{
		Value:      ValTypeI64(),
		Mutability: MutabilityVar,
	}, I64(0))
	if err != nil {
		t.Fatal(err)
	}
	defer global.Close()
	gt := global.Type()
	if !gt.Value.IsI64() || gt.Mutability != MutabilityVar {
		t.Fatalf("got %s/%s", gt.Value, gt.Mutability)
	}
}

func TestResourceConstructorsValidateArguments(t *testing.T) {
	if _, err := NewFunction(FunctionType{}, nil); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil host function: got %v, want ErrInvalidArgument", err)
	}
	var nilOption FunctionOption
	if _, err := NewFunction(FunctionType{}, func(*CallContext, []Value) ([]Value, error) {
		return nil, nil
	}, nilOption); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil function option: got %v, want ErrInvalidArgument", err)
	}

	if _, err := NewMemory(MemoryType{
		Limits: Limits{Min: 2, Max: 1, HasMax: true},
	}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid memory limits: got %v, want ErrInvalidArgument", err)
	}
	if _, err := NewMemory(MemoryType{
		Limits: Limits{Min: 1, Shared: true},
	}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("shared memory without maximum: got %v, want ErrInvalidArgument", err)
	}
	for name, limits := range map[string]Limits{
		"memory32 minimum": {Min: 1<<16 + 1},
		"memory32 maximum": {Max: 1<<16 + 1, HasMax: true},
		"memory64 minimum": {Min: 1<<48 + 1, Is64: true},
		"memory64 maximum": {Max: 1<<48 + 1, HasMax: true, Is64: true},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewMemory(MemoryType{Limits: limits}); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("got %v, want ErrInvalidArgument", err)
			}
		})
	}
	for name, limits := range map[string]Limits{
		"table32 minimum": {Min: 1 << 32},
		"table32 maximum": {Max: 1 << 32, HasMax: true},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewTable(TableType{
				Element: ValTypeFuncRef(),
				Limits:  limits,
			}); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("got %v, want ErrInvalidArgument", err)
			}
		})
	}
	if err := validateTableLimits(Limits{
		Min:    ^uint64(0),
		Max:    ^uint64(0),
		HasMax: true,
		Is64:   true,
	}); err != nil {
		t.Fatalf("valid table64 descriptor rejected: %v", err)
	}

	globalType := GlobalType{Value: ValTypeI64(), Mutability: MutabilityVar}
	if _, err := NewGlobal(globalType, I32(1)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("global initializer mismatch: got %v, want ErrInvalidArgument", err)
	}
	if _, err := NewGlobal(GlobalType{
		Value: ValTypeI64(), Mutability: Mutability(99),
	}, I64(1)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("invalid global mutability: got %v, want ErrInvalidArgument", err)
	}

	ref := NewExternRef("closed")
	value := ExternRefValue(ref)
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}
	externTable := TableType{Element: ValTypeExternRef(), Limits: Limits{Min: 1}}
	if _, err := NewTableWithInit(externTable, I32(1)); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("non-reference table initializer: got %v, want ErrInvalidArgument", err)
	}
	if _, err := NewTableWithInit(externTable, value); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed table initializer owner: got %v, want ErrClosed", err)
	}
	if _, err := NewGlobal(GlobalType{
		Value: ValTypeExternRef(), Mutability: MutabilityVar,
	}, value); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed global initializer owner: got %v, want ErrClosed", err)
	}
}

func TestImportExportTypes(t *testing.T) {
	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer loader.Close()

	hostCall, err := loader.LoadBytes(testwasm.HostCallModule())
	if err != nil {
		t.Fatal(err)
	}
	defer hostCall.Close()

	imports := hostCall.Imports()
	if len(imports) != 1 {
		t.Fatalf("imports: got %d, want 1", len(imports))
	}
	imp := imports[0]
	if imp.ModuleName() != "env" || imp.Name() != "host_add" {
		t.Fatalf("import name: got %q.%q, want env.host_add", imp.ModuleName(), imp.Name())
	}
	if imp.ExternalType() != ExternalTypeFunction {
		t.Fatalf("import type: got %s, want function", imp.ExternalType())
	}
	if ft, ok := imp.FunctionType(); !ok ||
		len(ft.Params) != 2 || !ft.Params[0].IsI32() || !ft.Params[1].IsI32() ||
		len(ft.Results) != 1 || !ft.Results[0].IsI32() {
		t.Fatal("unexpected host_add function type")
	}
	_, tableOK := imp.TableType()
	_, memoryOK := imp.MemoryType()
	_, globalOK := imp.GlobalType()
	_, tagOK := imp.TagType()
	if tableOK || memoryOK || globalOK || tagOK {
		t.Fatal("function import returned a non-function typed view")
	}

	exports := hostCall.Exports()
	if len(exports) != 1 {
		t.Fatalf("exports: got %d, want 1", len(exports))
	}
	exp := exports[0]
	if exp.Name() != "call_host" {
		t.Fatalf("export name: got %q, want call_host", exp.Name())
	}
	if exp.ExternalType() != ExternalTypeFunction {
		t.Fatalf("export type: got %s, want function", exp.ExternalType())
	}
	if _, ok := exp.FunctionType(); !ok {
		t.Fatal("function export returned a nil function type")
	}
	_, tableOK = exp.TableType()
	_, memoryOK = exp.MemoryType()
	_, globalOK = exp.GlobalType()
	_, tagOK = exp.TagType()
	if tableOK || memoryOK || globalOK || tagOK {
		t.Fatal("function export returned a non-function typed view")
	}

	memory, err := loader.LoadBytes(testwasm.MemoryModule())
	if err != nil {
		t.Fatal(err)
	}
	defer memory.Close()

	var memExport *ExportType
	for _, candidate := range memory.Exports() {
		if candidate.Name() == "mem" {
			memExport = candidate
			break
		}
	}
	if memExport == nil {
		t.Fatal("memory module has no mem export")
	}
	if memExport.ExternalType() != ExternalTypeMemory {
		t.Fatalf("mem export type: got %s, want memory", memExport.ExternalType())
	}
	mt, ok := memExport.MemoryType()
	if !ok {
		t.Fatal("memory export returned a nil memory type")
	}
	if got := mt.Limits; got != (Limits{Min: 1}) {
		t.Fatalf("mem export limits: got %+v, want {Min:1}", got)
	}
	_, functionOK := memExport.FunctionType()
	_, tableOK = memExport.TableType()
	_, globalOK = memExport.GlobalType()
	_, tagOK = memExport.TagType()
	if functionOK || tableOK || globalOK || tagOK {
		t.Fatal("memory export returned a non-memory typed view")
	}

	// Descriptors are independent copies, not views into the AST.
	if err := memory.Close(); err != nil {
		t.Fatal(err)
	}
	if mt.Limits.Min != 1 {
		t.Fatal("copied memory descriptor changed after closing its AST")
	}
}
