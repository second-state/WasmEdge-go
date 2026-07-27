package wasmedge_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	wasmedge "github.com/second-state/WasmEdge-go/v2"
	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

// The one-shot path: load, validate, instantiate and execute in one call.
func ExampleVM_RunBytes() {
	vm, err := wasmedge.NewVM(nil)
	if err != nil {
		panic(err)
	}
	defer vm.Close()

	out, err := vm.RunBytes(testwasm.FibModule(), "fib", wasmedge.I32(10))
	if err != nil {
		panic(err)
	}
	fmt.Println(out[0].I32())
	// Output: 89
}

// Host functions: a Go func becomes a WASM import via reflection.
func ExampleWrapFunc() {
	env := wasmedge.NewModule("env")
	defer env.Close()
	err := env.AddFunction("host_add",
		wasmedge.MustWrapFunc(func(a, b int32) int32 { return a + b }))
	if err != nil {
		panic(err)
	}

	vm, err := wasmedge.NewVM(nil)
	if err != nil {
		panic(err)
	}
	defer vm.Close()
	if err := vm.RegisterImport(env); err != nil {
		panic(err)
	}

	out, err := vm.RunBytes(testwasm.HostCallModule(), "call_host",
		wasmedge.I32(40), wasmedge.I32(2))
	if err != nil {
		panic(err)
	}
	fmt.Println(out[0].I32())
	// Output: 42
}

// Cancellation: runaway WASM code is interrupted through context.Context.
func ExampleVM_ExecuteContext() {
	vm, err := wasmedge.NewVM(nil)
	if err != nil {
		panic(err)
	}
	defer vm.Close()

	if err := vm.LoadBytes(testwasm.LoopModule()); err != nil {
		panic(err)
	}
	if err := vm.Validate(); err != nil {
		panic(err)
	}
	if err := vm.Instantiate(); err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = vm.ExecuteContext(ctx, "run")
	fmt.Println(errors.Is(err, context.DeadlineExceeded))
	// Output: true
}

// The explicit pipeline, for embedders that need each stage.
func ExampleExecutor_Invoke() {
	loader, err := wasmedge.NewLoader(nil)
	if err != nil {
		panic(err)
	}
	defer loader.Close()
	ast, err := loader.LoadBytes(testwasm.AddModule())
	if err != nil {
		panic(err)
	}
	defer ast.Close()

	validator, err := wasmedge.NewValidator(nil)
	if err != nil {
		panic(err)
	}
	defer validator.Close()
	if err := validator.Validate(ast); err != nil {
		panic(err)
	}

	exec, err := wasmedge.NewExecutor(nil)
	if err != nil {
		panic(err)
	}
	defer exec.Close()
	store := wasmedge.NewStore()
	defer store.Close()

	mod, err := exec.Instantiate(store, ast)
	if err != nil {
		panic(err)
	}
	defer mod.Close()

	add, _ := mod.Function("add")
	out, err := exec.Invoke(add, wasmedge.I32(19), wasmedge.I32(23))
	if err != nil {
		panic(err)
	}
	fmt.Println(out[0].I32())
	// Output: 42
}

// WASI is opt-in. Empty Preopens means the guest receives no filesystem
// preopens; arguments, environment, and standard streams are explicit.
func ExampleVM_WASIModule() {
	vm, err := wasmedge.NewVM(&wasmedge.Config{WASI: true})
	if err != nil {
		panic(err)
	}
	defer vm.Close()

	wasi, ok := vm.WASIModule()
	if !ok {
		panic("WASI module is unavailable")
	}
	if err := wasi.InitWASI(wasmedge.WASIConfig{
		Args:  []string{"guest"},
		Envs:  []string{"MODE=example"},
		Stdio: wasmedge.DiscardWASIStdio(),
		// No Preopens: the guest receives no preopened directories.
		// Discard stdio grants no ambient process streams.
	}); err != nil {
		panic(err)
	}

	if _, err := vm.RunBytes(testwasm.ProcExitModule(), "_start"); err != nil {
		panic(err)
	}
	code, err := wasi.WASIExitCode()
	if err != nil {
		panic(err)
	}
	fmt.Println(code)
	// Output: 7
}

// UnsafeSlice aliases engine-owned memory. It is useful only when avoiding a
// copy matters, and must be discarded before memory growth or owner teardown.
func ExampleMemory_UnsafeSlice() {
	memory, err := wasmedge.NewMemory(wasmedge.MemoryType{
		Limits: wasmedge.Limits{Min: 1},
	})
	if err != nil {
		panic(err)
	}
	defer memory.Close()

	if err := memory.Write(0, []byte("cat")); err != nil {
		panic(err)
	}
	view, err := memory.UnsafeSlice(0, 3)
	if err != nil {
		panic(err)
	}
	view[0] = 'b'

	copied, err := memory.Read(0, 3)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(copied))
	// Output: bat
}
