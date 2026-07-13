package wasmedge

import "testing"

func TestStore(t *testing.T) {
	store := NewStore()
	if store == nil {
		t.Fatal("NewStore() returned nil")
	}
	defer store.Release()

	if mods := store.ListModule(); len(mods) != 0 {
		t.Errorf("a new store should be empty, got %v", mods)
	}
	if store.FindModule("anything") != nil {
		t.Error("FindModule on an empty store should return nil")
	}

	executor := NewExecutor()
	defer executor.Release()
	ast := loadASTFromFile(t, testWasmPath("fib.wasm"))
	defer ast.Release()
	validator := NewValidator()
	defer validator.Release()
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	mod, err := executor.Register(store, ast, "math")
	if err != nil {
		t.Fatal(err)
	}
	defer mod.Release()

	mods := store.ListModule()
	if len(mods) != 1 || mods[0] != "math" {
		t.Errorf("ListModule() = %v, want [math]", mods)
	}
	if store.FindModule("math") == nil {
		t.Error("FindModule(math) should not return nil")
	}
}
