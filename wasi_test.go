package wasmedge

import (
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestWASIExitCodeViaVM(t *testing.T) {
	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	wasi, ok := vm.WASIModule()
	if !ok {
		t.Fatal("Config.WASI did not surface a WASI module")
	}

	// proc_exit(7) terminates gracefully: the run reports success and the
	// exit code is on the WASI module.
	if _, err := vm.RunBytes(testwasm.ProcExitModule(), "_start"); err != nil {
		t.Fatalf("proc_exit must surface as success, got %v", err)
	}
	if got := wasi.WASIExitCode(); got != 7 {
		t.Fatalf("exit code: %d, want 7", got)
	}
}

func TestWASIModuleStandalone(t *testing.T) {
	wasi := NewWASIModule(WASIConfig{
		Args: []string{"prog", "arg1"},
		Envs: []string{"K=V"},
	})
	if wasi == nil {
		t.Fatal("NewWASIModule failed")
	}
	defer wasi.Close()

	// The module exposes the WASI preview1 surface.
	if _, ok := wasi.Function("proc_exit"); !ok {
		t.Fatal("proc_exit missing from standalone WASI module")
	}
	if got := wasi.WASIExitCode(); got != 0 {
		t.Fatalf("fresh exit code: %d", got)
	}

	// Wire it into an explicit pipeline and run the fixture.
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	store := NewStore()
	defer store.Close()
	if err := exec.RegisterImport(store, wasi); err != nil {
		t.Fatal(err)
	}

	loader, _ := NewLoader(nil)
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.ProcExitModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()
	validator, _ := NewValidator(nil)
	defer validator.Close()
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	fn, _ := inst.Function("_start")
	if _, err := exec.Invoke(fn); err != nil {
		t.Fatalf("proc_exit must surface as success, got %v", err)
	}
	if got := wasi.WASIExitCode(); got != 7 {
		t.Fatalf("exit code: %d, want 7", got)
	}
	// Re-init (InitWASI) assertions arrive with intern task A9.
}

func TestPluginListing(t *testing.T) {
	// Plugin availability depends on the host; this only asserts the calls
	// are safe and consistent with each other.
	names := PluginNames()
	for _, n := range names {
		p, ok := FindPlugin(n)
		if !ok {
			t.Fatalf("listed plugin %q not findable", n)
		}
		if p.Name() != n {
			t.Fatalf("plugin name mismatch: %q vs %q", p.Name(), n)
		}
	}
	if _, ok := FindPlugin("definitely-not-a-plugin"); ok {
		t.Fatal("ghost plugin found")
	}
}
