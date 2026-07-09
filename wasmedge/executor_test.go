package wasmedge

import (
	"testing"
)

func TestExecutorInstantiate(t *testing.T) {
	store := NewStore()
	defer store.Release()
	executor := NewExecutor()
	if executor == nil {
		t.Fatal("NewExecutor() returned nil")
	}
	defer executor.Release()

	mod := instantiateFile(t, executor, store, testWasmPath("fib.wasm"))
	defer mod.Release()

	fn := mod.FindFunction("fib")
	if fn == nil {
		t.Fatal("exported function fib not found")
	}
	rets, err := executor.Invoke(fn, int32(10))
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if rets[0].(int32) != 89 {
		t.Errorf("fib(10) = %v, want 89", rets[0])
	}
}

func TestExecutorHostFunction(t *testing.T) {
	store := NewStore()
	defer store.Release()
	executor := NewExecutor()
	defer executor.Release()

	// Host function checking data passing and the calling frame.
	type hostData struct{ called bool }
	data := &hostData{}
	addfn := func(d interface{}, frame *CallingFrame, params []interface{}) ([]interface{}, Result) {
		d.(*hostData).called = true
		if frame.GetExecutor() == nil {
			t.Error("calling frame should expose the executor")
		}
		if frame.GetModule() == nil {
			t.Error("calling frame should expose the calling module")
		}
		if frame.GetMemoryByIndex(0) == nil {
			t.Error("calling module has a memory at index 0")
		}
		return []interface{}{params[0].(int32) + params[1].(int32)}, Result_Success
	}

	env := NewModule("env")
	defer env.Release()
	ftype := NewFunctionType(
		[]*ValType{NewValTypeI32(), NewValTypeI32()},
		[]*ValType{NewValTypeI32()})
	env.AddFunction("host_add", NewFunction(ftype, addfn, data, 0))
	ftype.Release()
	gtype := NewGlobalType(NewValTypeI32(), ValMut_Const)
	env.AddGlobal("host_glob", NewGlobal(gtype, int32(777)))
	gtype.Release()

	if err := executor.RegisterImport(store, env); err != nil {
		t.Fatalf("RegisterImport failed: %v", err)
	}

	mod := instantiateFile(t, executor, store, testWasmPath("types.wasm"))
	defer mod.Release()

	rets, err := executor.Invoke(mod.FindFunction("call_add"), int32(11), int32(22))
	if err != nil {
		t.Fatalf("call_add failed: %v", err)
	}
	if rets[0].(int32) != 33 {
		t.Errorf("call_add = %v, want 33", rets[0])
	}
	if !data.called {
		t.Error("host function should have been called with the bound data")
	}

	rets, err = executor.Invoke(mod.FindFunction("get_hglob"))
	if err != nil {
		t.Fatalf("get_hglob failed: %v", err)
	}
	if rets[0].(int32) != 777 {
		t.Errorf("get_hglob = %v, want 777", rets[0])
	}
}

func TestExecutorHostFunctionError(t *testing.T) {
	store := NewStore()
	defer store.Release()
	executor := NewExecutor()
	defer executor.Release()

	failfn := func(interface{}, *CallingFrame, []interface{}) ([]interface{}, Result) {
		return nil, NewResult(ErrCategory_UserLevel, 5566)
	}
	env := NewModule("env")
	defer env.Release()
	ftype := NewFunctionType(
		[]*ValType{NewValTypeI32(), NewValTypeI32()},
		[]*ValType{NewValTypeI32()})
	env.AddFunction("host_add", NewFunction(ftype, failfn, nil, 0))
	ftype.Release()
	gtype := NewGlobalType(NewValTypeI32(), ValMut_Const)
	env.AddGlobal("host_glob", NewGlobal(gtype, int32(0)))
	gtype.Release()

	if err := executor.RegisterImport(store, env); err != nil {
		t.Fatalf("RegisterImport failed: %v", err)
	}
	mod := instantiateFile(t, executor, store, testWasmPath("types.wasm"))
	defer mod.Release()

	_, err := executor.Invoke(mod.FindFunction("call_add"), int32(1), int32(2))
	if err == nil {
		t.Fatal("host function error should propagate")
	}
	res := err.(*Result)
	if res.GetErrorCategory() != ErrCategory_UserLevel {
		t.Errorf("error category = %d, want user-level", res.GetErrorCategory())
	}
	if res.GetCode() != 5566 {
		t.Errorf("error code = %d, want 5566", res.GetCode())
	}
}

func TestExecutorRegister(t *testing.T) {
	store := NewStore()
	defer store.Release()
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
		t.Fatalf("Register failed: %v", err)
	}
	defer mod.Release()

	found := store.FindModule("math")
	if found == nil {
		t.Fatal("registered module not found in the store")
	}
	rets, err := executor.Invoke(found.FindFunction("fib"), int32(5))
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
	if rets[0].(int32) != 8 {
		t.Errorf("fib(5) = %v, want 8", rets[0])
	}
}

func TestExecutorRegisterImportWithAlias(t *testing.T) {
	store := NewStore()
	defer store.Release()
	executor := NewExecutor()
	defer executor.Release()

	mod := NewModule("original_name")
	defer mod.Release()
	gtype := NewGlobalType(NewValTypeI32(), ValMut_Const)
	mod.AddGlobal("g", NewGlobal(gtype, int32(1)))
	gtype.Release()

	if err := executor.RegisterImportWithAlias(store, mod, "alias_name"); err != nil {
		t.Fatalf("RegisterImportWithAlias failed: %v", err)
	}
	if store.FindModule("alias_name") == nil {
		t.Error("module should be findable under the alias")
	}

	// Registering the same alias twice must fail with a name conflict.
	if err := executor.RegisterImportWithAlias(store, mod, "alias_name"); err == nil {
		t.Error("duplicate alias registration should fail")
	}
}

func TestExecutorAsyncInvoke(t *testing.T) {
	store := NewStore()
	defer store.Release()
	executor := NewExecutor()
	defer executor.Release()

	mod := instantiateFile(t, executor, store, testWasmPath("fib.wasm"))
	defer mod.Release()

	async := executor.AsyncInvoke(mod.FindFunction("fib"), int32(15))
	if async == nil {
		t.Fatal("AsyncInvoke returned nil")
	}
	defer async.Release()
	async.WaitFor(5000)
	rets, err := async.GetResult()
	if err != nil {
		t.Fatalf("async fib failed: %v", err)
	}
	if rets[0].(int32) != 987 {
		t.Errorf("fib(15) = %v, want 987", rets[0])
	}
}

func TestExecutorTrap(t *testing.T) {
	store := NewStore()
	defer store.Release()
	executor := NewExecutor()
	defer executor.Release()

	mod := instantiateFile(t, executor, store, testWasmPath("trap.wasm"))
	defer mod.Release()

	if _, err := executor.Invoke(mod.FindFunction("trap")); err == nil {
		t.Error("unreachable should trap")
	}
	if _, err := executor.Invoke(mod.FindFunction("div"), int32(1), int32(0)); err == nil {
		t.Error("division by zero should trap")
	}
	rets, err := executor.Invoke(mod.FindFunction("div"), int32(12), int32(3))
	if err != nil {
		t.Fatalf("div failed: %v", err)
	}
	if rets[0].(int32) != 4 {
		t.Errorf("div = %v, want 4", rets[0])
	}
}
