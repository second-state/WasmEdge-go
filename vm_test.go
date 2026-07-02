package wasmedge

import (
	"context"
	"errors"
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
	if len(fns) != 1 || fns[0].Name != "fib" || len(fns[0].Type.Parameters()) != 1 {
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

	vm.Reset()
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

func TestVMRegisterImport(t *testing.T) {
	vm, _ := NewVM(nil)
	defer vm.Close()

	env := NewModule("env")
	defer env.Close()
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
	// WaitFor coverage arrives with intern task A10.
}

func TestVMWithExternalStore(t *testing.T) {
	store := NewStore()
	defer store.Close()
	vm, err := NewVM(nil, WithExternalStore(store))
	if err != nil {
		t.Fatal(err)
	}
	defer vm.Close()

	if err := vm.RegisterModuleBytes("calc", testwasm.AddModule()); err != nil {
		t.Fatal(err)
	}
	if _, ok := store.Module("calc"); !ok {
		t.Fatal("registration did not reach the external store")
	}
}
