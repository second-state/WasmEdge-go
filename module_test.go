package wasmedge

import (
	"errors"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestHostModuleLifecycle(t *testing.T) {
	m := NewModuleWithData("env", map[string]int{"answer": 42})
	if m == nil {
		t.Fatal("NewModuleWithData failed")
	}
	if m.Name() != "env" {
		t.Fatalf("name: %q", m.Name())
	}
	if got := m.HostData().(map[string]int)["answer"]; got != 42 {
		t.Fatalf("host data: %d", got)
	}

	if err := m.AddFunction("f", MustWrapFunc(func() {})); err != nil {
		t.Fatal(err)
	}
	if got := m.FunctionNames(); len(got) != 1 || got[0] != "f" {
		t.Fatalf("function names: %v", got)
	}
	if _, ok := m.Function("f"); !ok {
		t.Fatal("lookup of added function failed")
	}
	if _, ok := m.Function("missing"); ok {
		t.Fatal("lookup of missing function succeeded")
	}

	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal("double Close must be a no-op")
	}
}

func TestModuleInstanceExports(t *testing.T) {
	loader, _ := NewLoader(nil)
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.MemoryModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()
	validator, _ := NewValidator(nil)
	defer validator.Close()
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	store := NewStore()
	defer store.Close()

	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	if names := inst.MemoryNames(); len(names) != 1 || names[0] != "mem" {
		t.Fatalf("memory names: %v", names)
	}
	if names := inst.FunctionNames(); len(names) != 1 || names[0] != "load8" {
		t.Fatalf("function names: %v", names)
	}
}

func TestStoreRegistry(t *testing.T) {
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	store := NewStore()
	defer store.Close()

	env := NewModule("env")
	defer env.Close()
	if err := exec.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	if err := exec.RegisterImportWithAlias(store, env, "env_alias"); err != nil {
		t.Fatal(err)
	}

	names := store.ModuleNames()
	if len(names) != 2 {
		t.Fatalf("module names: %v", names)
	}
	if _, ok := store.Module("env"); !ok {
		t.Fatal("env not found")
	}
	if _, ok := store.Module("env_alias"); !ok {
		t.Fatal("alias not found")
	}
	if _, ok := store.Module("ghost"); ok {
		t.Fatal("ghost module found")
	}

	// Registering the same name twice must fail cleanly.
	if err := exec.RegisterImport(store, env); err == nil {
		t.Fatal("duplicate registration must fail")
	}
}

func TestGlobalConstructor(t *testing.T) {
	gt := NewGlobalType(ValTypeI64(), MutabilityConst)
	defer gt.Close()
	g := NewGlobal(gt, I64(9))
	if g == nil {
		t.Fatal("NewGlobal failed")
	}
	defer g.Close()
	if !g.Type().ValType().IsI64() || g.Type().Mutability() != MutabilityConst {
		t.Fatal("global type mismatch")
	}
	// Value get/set assertions arrive with intern task A7.
}

func TestTableConstructorAndAccess(t *testing.T) {
	tt := NewTableType(ValTypeExternRef(), Limits{Min: 2, Max: 4, HasMax: true})
	defer tt.Close()
	tbl := NewTable(tt)
	if tbl == nil {
		t.Fatal("NewTable failed")
	}
	defer tbl.Close()

	ref := NewExternRef("slot-1")
	defer ref.Close()
	if err := tbl.Set(1, ExternRefValue(ref)); err != nil {
		t.Fatal(err)
	}
	got, err := tbl.Get(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.ExternRef() == nil || got.ExternRef().Value().(string) != "slot-1" {
		t.Fatalf("table round-trip lost payload: %v", got)
	}

	var we *Error
	if _, err := tbl.Get(99); !errors.As(err, &we) {
		t.Fatalf("out-of-bounds Get must return *Error, got %v", err)
	}
	// Size/Grow/Type assertions arrive with intern task A6.
}
