package wasmedge

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

func TestVMRunBytes(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	out, err := vm.RunBytes(testwasm.FibModule(), "fib", I32(10))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 89 {
		t.Fatalf("fib(10) = %v", out)
	}
}

func TestVMStagedWorkflow(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	// Executing before loading violates the VM workflow.
	if _, err := vm.Execute("fib", I32(1)); err == nil {
		t.Fatal("execute before load must fail")
	}

	if err := vm.LoadBytes(testwasm.FibModule()); err != nil {
		t.Fatal(err)
	}
	// Instantiate before Validate violates the workflow and reports it.
	var we *Error
	if err := vm.Instantiate(); !errors.As(err, &we) || we.Code != ErrCodeWrongVMWorkflow {
		t.Fatalf("want WrongVMWorkflow, got %v", err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}

	out, err := vm.Execute("fib", I32(21))
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != 17711 {
		t.Fatalf("fib(21) = %v", out)
	}

	fns := vm.Functions()
	if len(fns) != 1 || fns[0].Name != "fib" || len(fns[0].Type.Params) != 1 {
		t.Fatalf("functions: %+v", fns)
	}
	if vm.ActiveModule() == nil {
		t.Fatal("no active module after instantiate")
	}
	if _, ok := vm.FunctionType("fib"); !ok {
		t.Fatal("fib type lookup failed")
	}
	if _, ok := vm.FunctionType("nope"); ok {
		t.Fatal("ghost function type found")
	}

	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}
	if vm.ActiveModule() != nil {
		t.Fatal("active module survived Reset")
	}
}

func TestVMExecuteUnknownFunction(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()
	if err := vm.LoadBytes(testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}
	var we *Error
	_, err := vm.Execute("missing", I32(1))
	if !errors.As(err, &we) || we.Code != ErrCodeFuncNotFound {
		t.Fatalf("want FuncNotFound, got %v", err)
	}
}

func TestVMRegisterModule(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()

	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	// Plugins installed on the host auto-register their modules into every
	// VM, so assert membership rather than the exact registry contents.
	names := vm.RegisteredModuleNames()
	found := false
	for _, n := range names {
		found = found || n == "calc"
	}
	if !found {
		t.Fatalf("calc missing from registered modules: %v", names)
	}
	mod, ok := vm.RegisteredModule("calc")
	if !ok {
		t.Fatal("calc not found")
	}
	if got := mod.FunctionNames(); len(got) != 1 || got[0] != "add" {
		t.Fatalf("calc exports: %v", got)
	}
	// Duplicate names are rejected by the engine.
	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err == nil {
		t.Fatal("duplicate module name must fail")
	}
}

func TestVMFileAndRegisteredConveniences(t *testing.T) {
	path := filepath.Join(t.TempDir(), "add.wasm")
	if err := os.WriteFile(path, testwasm.AddModule(), 0o600); err != nil {
		t.Fatal(err)
	}

	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	if err := vm.RegisterModuleFile("calc", path); err != nil {
		t.Fatal(err)
	}
	ft, ok := vm.RegisteredFunctionType("calc", "add")
	if !ok || len(ft.Params) != 2 || len(ft.Results) != 1 {
		t.Fatalf("registered function type: %#v, ok=%v", ft, ok)
	}
	if _, ok := vm.RegisteredFunctionType("calc", "missing"); ok {
		t.Fatal("found a missing registered function")
	}

	out, err := vm.ExecuteRegistered("calc", "add", I32(20), I32(22))
	if err != nil || len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("ExecuteRegistered: out=%v err=%v", out, err)
	}
	ex, err := vm.ExecuteRegisteredAsync("calc", "add", I32(5), I32(6))
	if err != nil {
		t.Fatal(err)
	}
	out, err = ex.Wait()
	if err != nil || len(out) != 1 || out[0].I32() != 11 {
		t.Fatalf("ExecuteRegisteredAsync: out=%v err=%v", out, err)
	}
	out, err = vm.ExecuteRegisteredContext(
		context.Background(), "calc", "add", I32(7), I32(8))
	if err != nil || len(out) != 1 || out[0].I32() != 15 {
		t.Fatalf("ExecuteRegisteredContext: out=%v err=%v", out, err)
	}

	out, err = vm.RunFile(path, "add", I32(9), I32(10))
	if err != nil || len(out) != 1 || out[0].I32() != 19 {
		t.Fatalf("RunFile: out=%v err=%v", out, err)
	}
}

func TestVMRegisterImport(t *testing.T) {
	vm, _ := NewVM(nil)

	env := NewModule("env")
	t.Cleanup(func() {
		_ = vm.Close()
		_ = env.Close()
	})
	if err := env.AddFunction("host_add",
		MustWrapFunc(func(a, b int32) int32 { return a * b })); err != nil {
		t.Fatal(err)
	}
	if err := vm.RegisterImport(env); err != nil {
		t.Fatal(err)
	}

	out, err := vm.RunBytes(testwasm.HostCallModule(), "call_host", I32(6), I32(7))
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != 42 {
		t.Fatalf("out=%v", out)
	}
}

func TestVMRegisterImportWithAlias(t *testing.T) {
	vm, _ := NewVM(nil)

	host := NewModule("private-name")
	t.Cleanup(func() {
		_ = vm.Close()
		_ = host.Close()
	})
	if err := host.AddFunction("host_add",
		MustWrapFunc(func(a, b int32) int32 { return a * b })); err != nil {
		t.Fatal(err)
	}
	if err := vm.RegisterImportWithAlias(host, "env"); err != nil {
		t.Fatal(err)
	}
	if len(vm.imports) != 1 || vm.imports[0] != host {
		t.Fatal("VM did not retain the aliased import")
	}

	out, err := vm.RunBytes(testwasm.HostCallModule(), "call_host", I32(6), I32(7))
	if err != nil || len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("aliased host call: out=%v err=%v", out, err)
	}
}

func TestVMExecuteContextCancel(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()
	if _, err := vm.RunBytes(testwasm.AddModule(), "add", I32(1), I32(2)); err != nil {
		t.Fatal(err)
	}
	if err := vm.LoadBytes(testwasm.LoopModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := vm.ExecuteContext(ctx, "run")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("cancellation took %v", elapsed)
	}

	// The VM survives an interrupted execution.
	out, err := vm.RunBytes(testwasm.FibModule(), "fib", I32(5))
	if err != nil || out[0].I32() != 8 {
		t.Fatalf("VM unusable after cancel: %v %v", out, err)
	}
}

func TestVMExecuteContextAlreadyCanceled(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()
	if err := vm.LoadBytes(testwasm.LoopModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := vm.ExecuteContext(ctx, "run"); !errors.Is(err, context.Canceled) {
		t.Fatalf("want Canceled, got %v", err)
	}
}

func TestVMExecuteAsync(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()
	if _, err := vm.RunBytes(testwasm.FibModule(), "fib", I32(1)); err != nil {
		t.Fatal(err)
	}

	ex, err := vm.ExecuteAsync("fib", I32(15))
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("async execution did not lease its originating VM: %v", err)
	}
	out, err := ex.Wait()
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != 987 {
		t.Fatalf("fib(15) = %v", out)
	}
	if err := ex.Close(); err != nil {
		t.Fatal("Close after Wait must be a no-op")
	}
}

func TestVMExecuteAsyncWaitForAndCancel(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()
	if err := vm.LoadBytes(testwasm.LoopModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}

	ex, err := vm.ExecuteAsync("run")
	if err != nil {
		t.Fatal(err)
	}
	if ex.WaitFor(0) {
		t.Fatal("infinite loop unexpectedly completed")
	}
	ex.Cancel()
	_, err = ex.Wait()
	var we *Error
	if !errors.As(err, &we) || we.Code != ErrCodeInterrupted {
		t.Fatalf("cancel: want ErrCodeInterrupted, got %v", err)
	}
}

func TestVMWithExternalStore(t *testing.T) {
	store := NewStore()
	defer store.Close()
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	if vm.store != store {
		t.Fatal("VM did not retain its external Store")
	}

	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Module("calc"); !ok {
		t.Fatal("registration did not reach the external store")
	}
}

func TestVMExternalStoreLifetime(t *testing.T) {
	closed := NewStore()
	if err := closed.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := NewVM(nil, WithExternalStore(closed)); !errors.Is(err, ErrClosed) {
		t.Fatalf("closed external store: got %v, want ErrClosed", err)
	}

	store := NewStore()
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("close attached store: got %v, want ErrInUse", err)
	}
	if err := vm.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store after VM: %v", err)
	}
}

func TestVMBorrowedModuleViewsExpireWithGeneration(t *testing.T) {
	t.Run("active module after reinstantiate", func(t *testing.T) {
		vm, err := NewVM(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer vm.Close()

		instantiateVMFixture(t, vm, testwasm.AddModule())
		active := vm.ActiveModule()
		if active == nil {
			t.Fatal("active module missing")
		}
		add, ok := active.Function("add")
		if !ok {
			t.Fatal("add export missing")
		}

		instantiateVMFixture(t, vm, testwasm.FibModule())
		assertVMPanics(t, "replaced active module", func() { _ = active.FunctionNames() })
		assertVMPanics(t, "function from replaced active module", func() { _ = add.Type() })

		current := vm.ActiveModule()
		if current == nil {
			t.Fatal("replacement active module missing")
		}
		if _, ok := current.Function("fib"); !ok {
			t.Fatal("replacement active module has no fib export")
		}
	})

	t.Run("active module after reset", func(t *testing.T) {
		vm, err := NewVM(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer vm.Close()

		instantiateVMFixture(t, vm, testwasm.AddModule())
		active := vm.ActiveModule()
		add, ok := active.Function("add")
		if !ok {
			t.Fatal("add export missing")
		}
		if err := vm.Reset(); err != nil {
			t.Fatal(err)
		}
		assertVMPanics(t, "reset active module", func() { _ = active.Name() })
		assertVMPanics(t, "function from reset active module", func() { _ = add.Type() })
	})

	t.Run("registered and WASI modules after reset", func(t *testing.T) {
		vm, err := NewVM(&Config{WASI: true})
		if err != nil {
			t.Fatal(err)
		}
		defer vm.Close()

		if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
			t.Fatal(err)
		}
		registered, ok := vm.RegisteredModule("calc")
		if !ok {
			t.Fatal("registered module missing")
		}
		add, ok := registered.Function("add")
		if !ok {
			t.Fatal("registered add export missing")
		}
		wasi, ok := vm.WASIModule()
		if !ok {
			t.Fatal("WASI module missing")
		}
		procExit, ok := wasi.Function("proc_exit")
		if !ok {
			t.Fatal("WASI proc_exit export missing")
		}

		// Replacing only the active module must not invalidate registered
		// modules, including the VM-owned WASI import.
		instantiateVMFixture(t, vm, testwasm.FibModule())
		if _, ok := registered.Function("add"); !ok {
			t.Fatal("active reinstantiate invalidated a registered module")
		}
		_, _ = wasi.WASIExitCode()

		if err := vm.Reset(); err != nil {
			t.Fatal(err)
		}
		assertVMPanics(t, "reset registered module", func() { _ = registered.Name() })
		assertVMPanics(t, "function from reset registered module", func() { _ = add.Type() })
		assertVMPanics(t, "reset WASI module", func() { _, _ = wasi.WASIExitCode() })
		assertVMPanics(t, "function from reset WASI module", func() { _ = procExit.Type() })
	})
}

func TestVMOwnedAccessorsExpireWithVM(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	loader := vm.Loader()
	validator := vm.Validator()
	executor := vm.Executor()
	if loader == nil || validator == nil || executor == nil {
		t.Fatal("VM returned a nil owned subsystem")
	}

	ast, err := loader.LoadBytes(testwasm.AddModule())
	if err != nil {
		t.Fatal(err)
	}
	defer ast.Close()
	if err := validator.Validate(ast); err != nil {
		t.Fatal(err)
	}
	if err := vm.Close(); err != nil {
		t.Fatal(err)
	}

	assertVMPanics(t, "VM loader after VM close", func() {
		_, _ = loader.LoadBytes(testwasm.AddModule())
	})
	assertVMPanics(t, "VM validator after VM close", func() {
		_ = validator.Validate(ast)
	})
	assertVMPanics(t, "VM executor after VM close", func() {
		_, _ = executor.Invoke(nil)
	})
}

func TestVMExternalStoreResetRegistrationBoundary(t *testing.T) {
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

	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}

	// WasmEdge 0.17.1 VMCleanup clears every non-built-in registration from
	// the VM's store, including registrations that predated an external-store
	// VM. The Go retention graph must mirror that native behavior.
	if _, ok := store.Module("calc"); ok {
		t.Fatal("VM-owned registration survived Reset")
	}
	if _, ok := store.Module("env"); ok {
		t.Fatal("pre-existing external-store registration survived Reset")
	}
	if err := vm.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestVMExternalStoreCloseRegistrationBoundary(t *testing.T) {
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
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if err := vm.Close(); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Module("env"); !ok {
		t.Fatal("VM.Close removed a pre-existing external-store registration")
	}
	if _, ok := store.Module("calc"); ok {
		t.Fatal("VM-owned registration survived VM.Close")
	}
}

func TestVMLeasesExternalStoreImportsUsedByGuest(t *testing.T) {
	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer exec.Close()
	store := NewStore()
	defer store.Close()
	env := NewModule("env")
	t.Cleanup(func() { _ = env.Close() })
	if err := env.AddFunction(
		"host_add",
		MustWrapFunc(func(a, b int32) int32 { return a + b }),
	); err != nil {
		t.Fatal(err)
	}
	if err := exec.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	out, err := vm.RunBytes(testwasm.HostCallModule(), "call_host", I32(20), I32(22))
	if err != nil || len(out) != 1 || out[0].I32() != 42 {
		t.Fatalf("external-store import call: out=%v err=%v", out, err)
	}
	if err := env.Close(); !errors.Is(err, ErrInUse) {
		t.Fatalf("close import used by active VM module: got %v, want ErrInUse", err)
	}
	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}
	if err := env.Close(); err != nil {
		t.Fatalf("close import after VM.Reset: %v", err)
	}
}

func TestVMBorrowedStoreSharesRegistryAndGeneration(t *testing.T) {
	vm, err := NewVM(&Config{WASI: true})
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	store := vm.Store()
	oldWASI, ok := store.Module("wasi_snapshot_preview1")
	if !ok {
		t.Fatal("VM Store view did not expose the built-in WASI module")
	}
	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	module, ok := store.Module("calc")
	if !ok {
		t.Fatal("VM Store view did not observe registered module")
	}
	if err := vm.Reset(); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Module("calc"); ok {
		t.Fatal("VM Store retained module after Reset")
	}
	assertVMPanics(t, "module obtained through VM Store before Reset", func() {
		_ = module.Name()
	})
	assertVMPanics(t, "WASI module obtained through VM Store before Reset", func() {
		_, _ = oldWASI.WASIExitCode()
	})
	freshWASI, ok := store.Module("wasi_snapshot_preview1")
	if !ok {
		t.Fatal("VM Store view did not expose WASI after Reset")
	}
	if _, err := freshWASI.WASIExitCode(); err != nil {
		t.Fatalf("fresh VM Store WASI provenance: %v", err)
	}
}

func TestVMAsyncOneShotWorkflows(t *testing.T) {
	t.Run("bytes", func(t *testing.T) {
		vm, err := NewVM(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer vm.Close()

		execution, err := vm.RunBytesAsync(testwasm.FibModule(), "fib", I32(10))
		if err != nil {
			t.Fatal(err)
		}
		if err := vm.Close(); !errors.Is(err, ErrInUse) {
			t.Fatalf("RunBytesAsync did not lease VM: %v", err)
		}
		out, err := execution.Wait()
		if err != nil || len(out) != 1 || out[0].I32() != 89 {
			t.Fatalf("RunBytesAsync: out=%v err=%v", out, err)
		}
	})

	t.Run("file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "add.wasm")
		if err := os.WriteFile(path, testwasm.AddModule(), 0o600); err != nil {
			t.Fatal(err)
		}
		vm, err := NewVM(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer vm.Close()

		execution, err := vm.RunFileAsync(path, "add", I32(20), I32(22))
		if err != nil {
			t.Fatal(err)
		}
		out, err := execution.Wait()
		if err != nil || len(out) != 1 || out[0].I32() != 42 {
			t.Fatalf("RunFileAsync: out=%v err=%v", out, err)
		}
	})

	t.Run("AST", func(t *testing.T) {
		loader, err := NewLoader(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer loader.Close()
		ast, err := loader.LoadBytes(testwasm.FibModule())
		if err != nil {
			t.Fatal(err)
		}
		defer ast.Close()
		vm, err := NewVM(nil)
		if err != nil {
			t.Fatal(err)
		}
		defer vm.Close()

		execution, err := vm.RunAsync(ast, "fib", I32(15))
		if err != nil {
			t.Fatal(err)
		}
		if err := ast.Close(); !errors.Is(err, ErrInUse) {
			t.Fatalf("RunAsync did not lease AST: %v", err)
		}
		out, err := execution.Wait()
		if err != nil || len(out) != 1 || out[0].I32() != 987 {
			t.Fatalf("RunAsync: out=%v err=%v", out, err)
		}
		if err := ast.Close(); err != nil {
			t.Fatalf("AST remained leased after Wait: %v", err)
		}
	})
}

func TestVMRunContextCancellationReleasesOrigin(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err = vm.RunBytesContext(ctx, testwasm.LoopModule(), "run")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunBytesContext: want DeadlineExceeded, got %v", err)
	}

	out, err := vm.RunBytes(testwasm.FibModule(), "fib", I32(5))
	if err != nil || len(out) != 1 || out[0].I32() != 8 {
		t.Fatalf("VM unusable after canceled one-shot run: out=%v err=%v", out, err)
	}
}

func TestExecutionCloseCancelsWaitsAndReleasesOrigin(t *testing.T) {
	vm, err := NewVM(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()
	instantiateVMFixture(t, vm, testwasm.LoopModule())

	execution, err := vm.ExecuteAsync("run")
	if err != nil {
		t.Fatal(err)
	}
	if execution.WaitFor(0) {
		t.Fatal("infinite loop unexpectedly completed")
	}
	start := time.Now()
	if err := execution.Close(); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("Execution.Close took %v", elapsed)
	}
	if err := execution.Close(); err != nil {
		t.Fatalf("second Execution.Close: %v", err)
	}

	out, err := vm.RunBytes(testwasm.FibModule(), "fib", I32(5))
	if err != nil || len(out) != 1 || out[0].I32() != 8 {
		t.Fatalf("VM remained leased after Execution.Close: out=%v err=%v", out, err)
	}
}

func instantiateVMFixture(t *testing.T, vm *VM, wasm []byte) {
	t.Helper()
	if err := vm.LoadBytes(wasm); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}
}

func assertVMPanics(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s remained usable", name)
		}
	}()
	fn()
}
