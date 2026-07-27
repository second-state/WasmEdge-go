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

	replacement := MustWrapFunc(func() {})
	defer replacement.Close()
	if err := m.AddFunction("f", replacement); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("duplicate function: want ErrAlreadyExists, got %v", err)
	}
	if !replacement.life.alive() {
		t.Fatal("duplicate registration consumed the rejected function")
	}

	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil {
		t.Fatal("double Close must be a no-op")
	}
}

func TestNilHostDataAndExternRef(t *testing.T) {
	m := NewModuleWithData("nil-data", nil)
	if m == nil {
		t.Fatal("NewModuleWithData(nil) failed")
	}
	defer m.Close()
	if data := m.HostData(); data != nil {
		t.Fatalf("HostData = %#v, want nil", data)
	}
	if ref := NewExternRef(nil); ref != nil {
		t.Fatalf("NewExternRef(nil) = %#v, want nil", ref)
	}
	if value := ExternRefValue(NewExternRef(nil)); !value.IsNullRef() {
		t.Fatal("nil ExternRef did not produce a null externref")
	}
}

func TestOpaqueTokenProvenanceIsDomainSeparated(t *testing.T) {
	extern := NewExternRef("extern payload")
	if extern == nil {
		t.Fatal("NewExternRef failed")
	}
	externToken := extern.token()

	// The exact same pointer is valid in the externref registry but remains
	// foreign to module host data.
	foreignModule := newModuleWithDataToken("foreign-module-data", externToken)
	if foreignModule == nil {
		t.Fatal("raw module creation failed")
	}
	if got := foreignModule.HostData(); got != nil {
		t.Fatalf("externref token resolved as module data: %#v", got)
	}
	if err := foreignModule.Close(); err != nil {
		t.Fatal(err)
	}
	if got := extern.Value(); got != "extern payload" {
		t.Fatalf("foreign module finalizer released externref: %#v", got)
	}

	module := NewModuleWithData("module-data", "module payload")
	if module == nil {
		t.Fatal("NewModuleWithData failed")
	}
	moduleToken := module.hostDataToken()

	// And a real module-data token with that exact address remains foreign
	// to the externref registry.
	foreignValue := externRefValueFromToken(moduleToken, module)
	if got := foreignValue.ExternRef(); got != nil {
		t.Fatalf("module-data token resolved as externref: %#v", got)
	}
	if _, ok := moduleDataTokens.Load(moduleToken); !ok {
		t.Fatal("module data token was not registered")
	}
	if err := module.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := moduleDataTokens.Load(moduleToken); ok {
		t.Fatal("native module finalizer did not remove data token")
	}
	if err := module.Close(); err != nil {
		t.Fatalf("repeated module Close: %v", err)
	}

	if err := extern.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := externRefTokens.Load(externToken); ok {
		t.Fatal("ExternRef.Close did not remove token")
	}
	if err := extern.Close(); err != nil {
		t.Fatalf("repeated ExternRef.Close: %v", err)
	}
}

func TestHostModuleRejectsBorrowedOwnership(t *testing.T) {
	source := NewModule("source")
	defer source.Close()
	if err := source.AddFunction("f", MustWrapFunc(func() {})); err != nil {
		t.Fatal(err)
	}
	borrowedFunc, ok := source.Function("f")
	if !ok {
		t.Fatal("borrowed function not found")
	}

	target := NewModule("target")
	defer target.Close()
	if err := target.AddFunction("f", borrowedFunc); !errors.Is(err, ErrOwnership) {
		t.Fatalf("borrowed transfer: want ErrOwnership, got %v", err)
	}
	if _, exists := target.Function("f"); exists {
		t.Fatal("rejected borrowed function was added to target")
	}

	store := NewStore()
	defer store.Close()
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	if err := exec.RegisterImport(store, source); err != nil {
		t.Fatal(err)
	}
	borrowedModule, ok := store.Module("source")
	if !ok {
		t.Fatal("registered module not found")
	}
	rejected := MustWrapFunc(func() {})
	defer rejected.Close()
	if err := borrowedModule.AddFunction("g", rejected); !errors.Is(err, ErrOwnership) {
		t.Fatalf("mutating borrowed module: want ErrOwnership, got %v", err)
	}
}

func TestHostModuleReferenceTransferPreflight(t *testing.T) {
	referenceContainer := func(t *testing.T, roots *referenceRoots) any {
		t.Helper()
		if roots == nil {
			t.Fatal("borrowed reference object has no shared roots")
		}
		roots.mu.Lock()
		defer roots.mu.Unlock()
		return roots.container
	}

	t.Run("closed table", func(t *testing.T) {
		target := NewModule("closed-table-target")
		defer target.Close()
		table, err := NewTable(TableType{
			Element: ValTypeExternRef(),
			Limits:  Limits{Min: 1},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := table.Close(); err != nil {
			t.Fatal(err)
		}
		if err := target.AddTable("closed", table); !errors.Is(err, ErrClosed) {
			t.Fatalf("closed table: want ErrClosed, got %v", err)
		}
		if _, ok := target.Table("closed"); ok {
			t.Fatal("closed table was added")
		}
	})

	t.Run("borrowed table", func(t *testing.T) {
		source := NewModule("borrowed-table-source")
		defer source.Close()
		target := NewModule("borrowed-table-target")
		defer target.Close()
		table, err := NewTable(TableType{
			Element: ValTypeExternRef(),
			Limits:  Limits{Min: 1},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := source.AddTable("table", table); err != nil {
			t.Fatal(err)
		}
		view, ok := source.Table("table")
		if !ok {
			t.Fatal("borrowed table not found")
		}
		before := referenceContainer(t, view.roots)
		if err := target.AddTable("borrowed", view); !errors.Is(err, ErrOwnership) {
			t.Fatalf("borrowed table: want ErrOwnership, got %v", err)
		}
		if after := referenceContainer(t, view.roots); after != before {
			t.Fatalf("rejected borrowed table rebound shared roots: before=%p after=%p", before, after)
		}
		if got := view.Size(); got != 1 {
			t.Fatalf("borrowed table unusable after rejection: size=%d", got)
		}
	})

	t.Run("closed global", func(t *testing.T) {
		target := NewModule("closed-global-target")
		defer target.Close()
		global, err := NewGlobal(GlobalType{
			Value: ValTypeI32(), Mutability: MutabilityVar,
		}, I32(1))
		if err != nil {
			t.Fatal(err)
		}
		if err := global.Close(); err != nil {
			t.Fatal(err)
		}
		if err := target.AddGlobal("closed", global); !errors.Is(err, ErrClosed) {
			t.Fatalf("closed global: want ErrClosed, got %v", err)
		}
		if _, ok := target.Global("closed"); ok {
			t.Fatal("closed global was added")
		}
	})

	t.Run("borrowed global", func(t *testing.T) {
		source := NewModule("borrowed-global-source")
		defer source.Close()
		target := NewModule("borrowed-global-target")
		defer target.Close()
		global, err := NewGlobal(GlobalType{
			Value: ValTypeI32(), Mutability: MutabilityVar,
		}, I32(7))
		if err != nil {
			t.Fatal(err)
		}
		if err := source.AddGlobal("global", global); err != nil {
			t.Fatal(err)
		}
		view, ok := source.Global("global")
		if !ok {
			t.Fatal("borrowed global not found")
		}
		before := referenceContainer(t, view.roots)
		if err := target.AddGlobal("borrowed", view); !errors.Is(err, ErrOwnership) {
			t.Fatalf("borrowed global: want ErrOwnership, got %v", err)
		}
		if after := referenceContainer(t, view.roots); after != before {
			t.Fatalf("rejected borrowed global rebound shared roots: before=%p after=%p", before, after)
		}
		if got := view.Value().I32(); got != 7 {
			t.Fatalf("borrowed global unusable after rejection: value=%d", got)
		}
	})
}

func TestModuleInvocationReferenceRoots(t *testing.T) {
	newReceiver := func(t *testing.T, module *Module) *Function {
		t.Helper()
		receiver := mustNewHostFunction(t,
			FunctionType{Params: []ValType{ValTypeFuncRef()}},
			func(*CallContext, []Value) ([]Value, error) { return nil, nil },
		)
		if err := module.AddFunction("receiver", receiver); err != nil {
			t.Fatal(err)
		}
		view, ok := module.Function("receiver")
		if !ok {
			t.Fatal("receiver function not found")
		}
		return view
	}
	newTarget := func(t *testing.T, module *Module) *Function {
		t.Helper()
		target := MustWrapFunc(func() {})
		if err := module.AddFunction("target", target); err != nil {
			t.Fatal(err)
		}
		view, ok := module.Function("target")
		if !ok {
			t.Fatal("target function not found")
		}
		return view
	}

	t.Run("same module", func(t *testing.T) {
		module := NewModule("same-invocation-root")
		receiver := newReceiver(t, module)
		target := newTarget(t, module)
		exec, err := NewExecutor(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer exec.Close()
		if _, err := exec.Invoke(receiver, FuncRefValue(target)); err != nil {
			t.Fatal(err)
		}
		if err := module.Close(); err != nil {
			t.Fatalf("module retained a lease on itself: %v", err)
		}
	})

	t.Run("external module", func(t *testing.T) {
		module := NewModule("external-invocation-root")
		receiver := newReceiver(t, module)
		external := NewModule("external-invocation-target")
		target := newTarget(t, external)
		exec, err := NewExecutor(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer exec.Close()
		if _, err := exec.Invoke(receiver, FuncRefValue(target)); err != nil {
			t.Fatal(err)
		}
		if err := external.Close(); !errors.Is(err, ErrInUse) {
			t.Fatalf("external module was not retained: %v", err)
		}
		if err := module.Close(); err != nil {
			t.Fatalf("close retaining module: %v", err)
		}
		if err := external.Close(); err != nil {
			t.Fatalf("external module lease was not released: %v", err)
		}
	})
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

	env := NewModule("env")
	t.Cleanup(func() {
		_ = store.Close()
		_ = env.Close()
	})
	if err := exec.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	if err := exec.RegisterImportWithAlias(store, env, "env_alias"); err != nil {
		t.Fatal(err)
	}
	if got := store.state.retained; len(got) != 2 || got[0] != env || got[1] != env {
		t.Fatalf("store did not retain registered module: %#v", got)
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
	g, err := NewGlobal(GlobalType{
		Value: ValTypeI64(), Mutability: MutabilityConst,
	}, I64(9))
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if !g.Type().Value.IsI64() || g.Type().Mutability != MutabilityConst {
		t.Fatal("global type mismatch")
	}
}

func TestTableConstructorAndAccess(t *testing.T) {
	tbl, err := NewTable(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 2, Max: 4, HasMax: true},
	})
	if err != nil {
		t.Fatal(err)
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
}

func TestModuleRootsStoredGoReferences(t *testing.T) {
	m := NewModule("refs")
	if m == nil {
		t.Fatal("NewModule failed")
	}

	table, err := NewTable(TableType{
		Element: ValTypeExternRef(),
		Limits:  Limits{Min: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	tableRef := NewExternRef("table")
	if err := table.Set(0, ExternRefValue(tableRef)); err != nil {
		t.Fatal(err)
	}
	if err := m.AddTable("refs", table); err != nil {
		t.Fatal(err)
	}
	if err := table.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tableRef.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("transferred table did not retain its reference: %v", err)
	}

	globalRef := NewExternRef("global")
	global, err := NewGlobal(GlobalType{
		Value: ValTypeExternRef(), Mutability: MutabilityVar,
	}, ExternRefValue(globalRef))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AddGlobal("current", global); err != nil {
		t.Fatal(err)
	}

	replacement := NewExternRef("replacement")
	globalView, ok := m.Global("current")
	if !ok {
		t.Fatal("added global not found")
	}
	if err := globalView.SetValue(ExternRefValue(replacement)); err != nil {
		t.Fatal(err)
	}
	if got := globalView.Value().ExternRef().Value(); got != "replacement" {
		t.Fatalf("global reference = %v", got)
	}

	if err := replacement.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("module global did not retain replacement: %v", err)
	}
	if err := globalRef.Close(); err != nil {
		t.Fatalf("overwritten global retained stale reference: %v", err)
	}

	// Destroy native containers before releasing any cgo.Handle they can
	// still expose.
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	if err := tableRef.Close(); err != nil {
		t.Fatal(err)
	}
	if err := replacement.Close(); err != nil {
		t.Fatal(err)
	}
}
