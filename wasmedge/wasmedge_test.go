package wasmedge

import (
	"path/filepath"
	"testing"
)

// testWasmPath returns the path of a fixture assembled from the *.wat
// sources in testdata with wat2wasm.
func testWasmPath(name string) string {
	return filepath.Join("testdata", name)
}

// loadASTFromFile loads and returns the AST of a fixture, failing the test on
// any error.
func loadASTFromFile(t *testing.T, path string) *AST {
	t.Helper()
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	defer loader.Release()
	ast, err := loader.LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(%q) failed: %v", path, err)
	}
	return ast
}

// buildEnvModule creates the host module matching the imports of
// testdata/types.wasm: the env.host_add function and env.host_glob global.
func buildEnvModule(t *testing.T, globVal int32) *Module {
	t.Helper()
	mod := NewModule("env")
	if mod == nil {
		t.Fatal("NewModule() returned nil")
	}

	addfn := func(data interface{}, callframe *CallingFrame, params []interface{}) ([]interface{}, Result) {
		return []interface{}{params[0].(int32) + params[1].(int32)}, Result_Success
	}
	ftype := NewFunctionType(
		[]*ValType{NewValTypeI32(), NewValTypeI32()},
		[]*ValType{NewValTypeI32()})
	hostfn := NewFunction(ftype, addfn, nil, 0)
	ftype.Release()
	if hostfn == nil {
		t.Fatal("NewFunction() returned nil")
	}
	mod.AddFunction("host_add", hostfn)

	gtype := NewGlobalType(NewValTypeI32(), ValMut_Const)
	glob := NewGlobal(gtype, globVal)
	gtype.Release()
	if glob == nil {
		t.Fatal("NewGlobal() returned nil")
	}
	mod.AddGlobal("host_glob", glob)

	return mod
}

// instantiateFile loads, validates, and instantiates a WASM file into the
// given store, returning the module instance.
func instantiateFile(t *testing.T, executor *Executor, store *Store, path string) *Module {
	t.Helper()
	ast := loadASTFromFile(t, path)
	defer ast.Release()
	validator := NewValidator()
	defer validator.Release()
	if err := validator.Validate(ast); err != nil {
		t.Fatalf("Validate(%q) failed: %v", path, err)
	}
	mod, err := executor.Instantiate(store, ast)
	if err != nil {
		t.Fatalf("Instantiate(%q) failed: %v", path, err)
	}
	return mod
}
