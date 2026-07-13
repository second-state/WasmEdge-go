package wasmedge

import (
	"os"
	"testing"
)

func TestLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	defer loader.Release()

	ast, err := loader.LoadFile(testWasmPath("fib.wasm"))
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	ast.Release()

	buf, err := os.ReadFile(testWasmPath("fib.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	ast, err = loader.LoadBuffer(buf)
	if err != nil {
		t.Fatalf("LoadBuffer failed: %v", err)
	}
	ast.Release()

	if _, err := loader.LoadFile("testdata/no_such_file.wasm"); err == nil {
		t.Error("loading a missing file should fail")
	}
	if _, err := loader.LoadBuffer([]byte{0x00, 0x01, 0x02, 0x03}); err == nil {
		t.Error("loading malformed bytes should fail")
	}
}

func TestValidator(t *testing.T) {
	validator := NewValidator()
	if validator == nil {
		t.Fatal("NewValidator() returned nil")
	}
	defer validator.Release()

	ast := loadASTFromFile(t, testWasmPath("fib.wasm"))
	defer ast.Release()
	if err := validator.Validate(ast); err != nil {
		t.Errorf("Validate failed on a valid module: %v", err)
	}

	// invalid.wasm is well-formed but fails validation (missing return value).
	badAST := loadASTFromFile(t, testWasmPath("invalid.wasm"))
	defer badAST.Release()
	if err := validator.Validate(badAST); err == nil {
		t.Error("Validate should fail on an invalid module")
	}
}

func TestASTListImports(t *testing.T) {
	ast := loadASTFromFile(t, testWasmPath("types.wasm"))
	defer ast.Release()

	imports := ast.ListImports()
	if len(imports) != 2 {
		t.Fatalf("expected 2 imports, got %d", len(imports))
	}

	byName := map[string]*ImportType{}
	for _, imp := range imports {
		if imp.GetModuleName() != "env" {
			t.Errorf("import module name = %q, want \"env\"", imp.GetModuleName())
		}
		byName[imp.GetExternalName()] = imp
	}

	fn, ok := byName["host_add"]
	if !ok {
		t.Fatal("import host_add not found")
	}
	if fn.GetExternalType() != ExternType_Function {
		t.Error("host_add should be a function import")
	}
	ftype := fn.GetExternalValue().(*FunctionType)
	if ftype.GetParametersLength() != 2 || ftype.GetReturnsLength() != 1 {
		t.Errorf("host_add type mismatch: %d params, %d returns",
			ftype.GetParametersLength(), ftype.GetReturnsLength())
	}
	params := ftype.GetParameters()
	if len(params) != 2 || !params[0].IsI32() || !params[1].IsI32() {
		t.Error("host_add parameters should be [i32, i32]")
	}
	returns := ftype.GetReturns()
	if len(returns) != 1 || !returns[0].IsI32() {
		t.Error("host_add returns should be [i32]")
	}

	glob, ok := byName["host_glob"]
	if !ok {
		t.Fatal("import host_glob not found")
	}
	if glob.GetExternalType() != ExternType_Global {
		t.Error("host_glob should be a global import")
	}
	gtype := glob.GetExternalValue().(*GlobalType)
	if !gtype.GetValType().IsI32() || gtype.GetMutability() != ValMut_Const {
		t.Error("host_glob should be a const i32 global")
	}
}

func TestASTListExports(t *testing.T) {
	ast := loadASTFromFile(t, testWasmPath("types.wasm"))
	defer ast.Release()

	exports := ast.ListExports()
	byName := map[string]*ExportType{}
	for _, exp := range exports {
		byName[exp.GetExternalName()] = exp
	}
	if len(byName) != 6 {
		t.Fatalf("expected 6 exports, got %d", len(byName))
	}

	for _, name := range []string{"add", "call_add", "get_hglob"} {
		exp, ok := byName[name]
		if !ok || exp.GetExternalType() != ExternType_Function {
			t.Errorf("export %q should be a function", name)
		}
		if _, ok := exp.GetExternalValue().(*FunctionType); !ok {
			t.Errorf("export %q external value should be a FunctionType", name)
		}
	}

	tab := byName["tab"]
	if tab == nil || tab.GetExternalType() != ExternType_Table {
		t.Fatal("export tab should be a table")
	}
	ttype := tab.GetExternalValue().(*TableType)
	if !ttype.GetRefType().IsFuncRef() {
		t.Error("tab should be a funcref table")
	}
	lim := ttype.GetLimit()
	if lim.GetMin() != 5 || !lim.HasMax() || lim.GetMax() != 20 || lim.Is64Bit() {
		t.Errorf("tab limit mismatch: %+v", lim)
	}

	mem := byName["mem"]
	if mem == nil || mem.GetExternalType() != ExternType_Memory {
		t.Fatal("export mem should be a memory")
	}
	mtype := mem.GetExternalValue().(*MemoryType)
	lim = mtype.GetLimit()
	if lim.GetMin() != 1 || !lim.HasMax() || lim.GetMax() != 2 || lim.IsShared() {
		t.Errorf("mem limit mismatch: %+v", lim)
	}

	glob := byName["glob"]
	if glob == nil || glob.GetExternalType() != ExternType_Global {
		t.Fatal("export glob should be a global")
	}
	gtype := glob.GetExternalValue().(*GlobalType)
	if !gtype.GetValType().IsI64() || gtype.GetMutability() != ValMut_Var {
		t.Error("glob should be a mutable i64 global")
	}
}

func TestFunctionTypeCreate(t *testing.T) {
	ftype := NewFunctionType(
		[]*ValType{NewValTypeI32(), NewValTypeF64()},
		[]*ValType{NewValTypeV128()})
	if ftype == nil {
		t.Fatal("NewFunctionType() returned nil")
	}
	defer ftype.Release()

	params := ftype.GetParameters()
	if len(params) != 2 || !params[0].IsI32() || !params[1].IsF64() {
		t.Error("parameter types mismatch")
	}
	returns := ftype.GetReturns()
	if len(returns) != 1 || !returns[0].IsV128() {
		t.Error("return types mismatch")
	}

	empty := NewFunctionType(nil, nil)
	if empty == nil {
		t.Fatal("NewFunctionType(nil, nil) returned nil")
	}
	defer empty.Release()
	if empty.GetParametersLength() != 0 || empty.GetReturnsLength() != 0 {
		t.Error("empty function type should have no parameters and returns")
	}
	if empty.GetParameters() != nil || empty.GetReturns() != nil {
		t.Error("empty function type should return nil slices")
	}
}

func TestTableTypeCreate(t *testing.T) {
	cases := []*Limit{
		NewLimit(10),
		NewLimitWithMax(10, 30),
		NewLimit64(10),
		NewLimit64WithMax(10, 30),
	}
	for i, lim := range cases {
		ttype := NewTableType(NewValTypeFuncRef(), lim)
		if ttype == nil {
			t.Fatalf("NewTableType() returned nil for case %d", i)
		}
		if !ttype.GetRefType().IsFuncRef() {
			t.Error("table ref type should be funcref")
		}
		if !ttype.GetLimit().IsEqual(lim) {
			t.Errorf("table limit round-trip mismatch for case %d", i)
		}
		ttype.Release()
		lim.Release()
	}

	elim := NewLimitWithMax(1, 2)
	etype := NewTableType(NewValTypeExternRef(), elim)
	elim.Release()
	if etype == nil {
		t.Fatal("externref table type creation failed")
	}
	if !etype.GetRefType().IsExternRef() {
		t.Error("table ref type should be externref")
	}
	etype.Release()
}

func TestMemoryTypeCreate(t *testing.T) {
	cases := []*Limit{
		NewLimit(1),
		NewLimitWithMax(1, 4),
		NewLimitSharedWithMax(1, 4),
		NewLimit64(1),
		NewLimit64WithMax(1, 4),
	}
	for i, lim := range cases {
		mtype := NewMemoryType(lim)
		if mtype == nil {
			t.Fatalf("NewMemoryType() returned nil for case %d", i)
		}
		if !mtype.GetLimit().IsEqual(lim) {
			t.Errorf("memory limit round-trip mismatch for case %d", i)
		}
		mtype.Release()
		lim.Release()
	}
}

func TestGlobalTypeCreate(t *testing.T) {
	gtype := NewGlobalType(NewValTypeF32(), ValMut_Var)
	if gtype == nil {
		t.Fatal("NewGlobalType() returned nil")
	}
	defer gtype.Release()
	if !gtype.GetValType().IsF32() {
		t.Error("global value type should be f32")
	}
	if gtype.GetMutability() != ValMut_Var {
		t.Error("global should be mutable")
	}
}
