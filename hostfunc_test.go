package wasmedge

import (
	"errors"
	"fmt"
	"testing"

	"github.com/second-state/WasmEdge-go/v2/internal/testwasm"
)

// instantiateWithHost loads the HostCall fixture, registers an "env" module
// providing host_add, and returns the instantiated module plus everything
// needed to invoke it. The cleanup order matters and doubles as lifecycle
// documentation: instance and env module before executor/store.
func instantiateWithHost(t *testing.T, hostAdd *Function) (*Executor, *Module) {
	t.Helper()

	env := NewModule("env")
	if env == nil {
		t.Fatal("NewModule failed")
	}
	if err := env.AddFunction("host_add", hostAdd); err != nil {
		t.Fatal(err)
	}

	loader, err := NewLoader(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { loader.Close() })

	ast, err := loader.LoadBytes(testwasm.HostCallModule())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ast.Close() })

	val, err := NewValidator(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { val.Close() })
	if err := val.Validate(ast); err != nil {
		t.Fatal(err)
	}

	exec, err := NewExecutor(nil)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	if err := exec.RegisterImport(store, env); err != nil {
		t.Fatal(err)
	}
	inst, err := exec.Instantiate(store, ast)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		inst.Close()
		env.Close()
		store.Close()
		exec.Close()
	})
	return exec, inst
}

func invokeCallHost(t *testing.T, exec *Executor, inst *Module, a, b int32) ([]Value, error) {
	t.Helper()
	fn, ok := inst.Function("call_host")
	if !ok {
		t.Fatal("call_host export missing")
	}
	return exec.Invoke(fn, I32(a), I32(b))
}

func TestHostFunctionRoundTrip(t *testing.T) {
	var sawCtx bool
	add := NewFunction(
		NewFunctionType([]ValType{ValTypeI32(), ValTypeI32()}, []ValType{ValTypeI32()}),
		func(call *CallContext, params []Value) ([]Value, error) {
			sawCtx = call != nil && call.Executor() != nil && call.Module() != nil
			return []Value{I32(params[0].I32() + params[1].I32())}, nil
		})
	if add == nil {
		t.Fatal("NewFunction failed")
	}

	exec, inst := instantiateWithHost(t, add)
	out, err := invokeCallHost(t, exec, inst, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].I32() != 7 {
		t.Fatalf("out=%v", out)
	}
	if !sawCtx {
		t.Fatal("CallContext was not fully populated during the host call")
	}
}

func TestWrapFunc(t *testing.T) {
	add, err := WrapFunc(func(a, b int32) int32 { return a + b })
	if err != nil {
		t.Fatal(err)
	}
	exec, inst := instantiateWithHost(t, add)
	out, err := invokeCallHost(t, exec, inst, 20, 22)
	if err != nil {
		t.Fatal(err)
	}
	if out[0].I32() != 42 {
		t.Fatalf("out=%v", out)
	}
}

func TestWrapFuncSignatures(t *testing.T) {
	// Signature derivation only; invocation is covered elsewhere.
	cases := []struct {
		fn       any
		nParams  int
		nResults int
	}{
		{func() {}, 0, 0},
		{func(int32) int64 { return 0 }, 1, 1},
		{func(*CallContext, uint64, float32) (float64, error) { return 0, nil }, 2, 1},
		{func(uint32) error { return nil }, 1, 0},
	}
	for i, c := range cases {
		f, err := WrapFunc(c.fn)
		if err != nil {
			t.Fatalf("case %d: %v", i, err)
		}
		ft := f.Type()
		if len(ft.Parameters()) != c.nParams || len(ft.Results()) != c.nResults {
			t.Errorf("case %d: got %d/%d params/results, want %d/%d",
				i, len(ft.Parameters()), len(ft.Results()), c.nParams, c.nResults)
		}
		f.Close()
	}

	if _, err := WrapFunc(42); err == nil {
		t.Error("non-func must be rejected")
	}
	if _, err := WrapFunc(func(string) {}); err == nil {
		t.Error("string parameter must be rejected")
	}
	if _, err := WrapFunc(func(...int32) {}); err == nil {
		t.Error("variadic func must be rejected")
	}
}

func TestHostFunctionPanicIsContained(t *testing.T) {
	boom := MustWrapFunc(func(a, b int32) int32 { panic("boom") })
	exec, inst := instantiateWithHost(t, boom)
	_, err := invokeCallHost(t, exec, inst, 1, 2)
	var we *Error
	if !errors.As(err, &we) {
		t.Fatalf("want *Error from contained panic, got %v", err)
	}
	if we.Category != ErrCategoryUserLevel {
		t.Fatalf("want user-level category, got %+v", we)
	}
	// The process surviving to this line is the real assertion.
}

func TestHostFunctionErrorRoundTrip(t *testing.T) {
	custom := &Error{Category: ErrCategoryUserLevel, Code: 0xBEEF}
	fail := NewFunction(
		NewFunctionType([]ValType{ValTypeI32(), ValTypeI32()}, []ValType{ValTypeI32()}),
		func(*CallContext, []Value) ([]Value, error) { return nil, custom })
	exec, inst := instantiateWithHost(t, fail)
	_, err := invokeCallHost(t, exec, inst, 1, 2)
	if !errors.Is(err, custom) {
		t.Fatalf("user error code lost through the engine: got %v", err)
	}
}

func TestHostFunctionTerminate(t *testing.T) {
	stop := NewFunction(
		NewFunctionType([]ValType{ValTypeI32(), ValTypeI32()}, []ValType{ValTypeI32()}),
		func(*CallContext, []Value) ([]Value, error) { return nil, Terminate })
	exec, inst := instantiateWithHost(t, stop)
	if _, err := invokeCallHost(t, exec, inst, 1, 2); err != nil {
		t.Fatalf("Terminate must surface as success, got %v", err)
	}
}

func TestCallContextMustNotEscape(t *testing.T) {
	var escaped *CallContext
	leak := NewFunction(
		NewFunctionType([]ValType{ValTypeI32(), ValTypeI32()}, []ValType{ValTypeI32()}),
		func(call *CallContext, params []Value) ([]Value, error) {
			escaped = call
			return []Value{I32(0)}, nil
		})
	exec, inst := instantiateWithHost(t, leak)
	if _, err := invokeCallHost(t, exec, inst, 1, 2); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("using an escaped CallContext must panic")
		}
	}()
	_ = escaped.Executor()
}

func TestFunctionTransferSemantics(t *testing.T) {
	f := MustWrapFunc(func(a, b int32) int32 { return 0 })
	env := NewModule("env")
	defer env.Close()
	if err := env.AddFunction("host_add", f); err != nil {
		t.Fatal(err)
	}
	// After transfer the wrapper is inert: Close must not double-free.
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	// Adding a closed/transferred function is rejected in Go, not in C.
	if err := env.AddFunction("again", f); err == nil {
		t.Fatal("adding a transferred function must fail")
	}
}

func TestExternRefThroughHostFunction(t *testing.T) {
	type box struct{ s string }
	payload := &box{s: "hi"}
	ref := NewExternRef(payload)
	defer ref.Close()

	// The host function receives i32s from WASM here; externref transport
	// is exercised directly through values until a table fixture exists.
	val := ExternRefValue(ref)
	if got := val.ExternRef().Value().(*box); got != payload {
		t.Fatal("externref payload identity lost")
	}
	_ = fmt.Sprintf("%v", val) // String() must not panic on refs
}
