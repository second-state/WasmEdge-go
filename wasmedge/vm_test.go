package wasmedge

import (
	"os"
	"strings"
	"testing"
)

// asyncLoopWasm is a module exporting "_start" which loops forever. It is used
// for testing the cancellation of asynchronous executions.
var asyncLoopWasm = []byte{
	0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x01, 0x04, 0x01, 0x60,
	0x00, 0x00, 0x03, 0x02, 0x01, 0x00, 0x05, 0x03, 0x01, 0x00, 0x01, 0x07,
	0x0a, 0x01, 0x06, 0x5f, 0x73, 0x74, 0x61, 0x72, 0x74, 0x00, 0x00, 0x0a,
	0x09, 0x01, 0x07, 0x00, 0x03, 0x40, 0x0c, 0x00, 0x0b, 0x0b,
}

func TestVMRunWasm(t *testing.T) {
	vm := NewVM()
	if vm == nil {
		t.Fatal("NewVM() returned nil")
	}
	defer vm.Release()

	rets, err := vm.RunWasmFile(testWasmPath("fib.wasm"), "fib", int32(10))
	if err != nil {
		t.Fatalf("RunWasmFile failed: %v", err)
	}
	if rets[0].(int32) != 89 {
		t.Errorf("fib(10) = %v, want 89", rets[0])
	}

	buf, err := os.ReadFile(testWasmPath("fib.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	rets, err = vm.RunWasmBuffer(buf, "fib", int32(11))
	if err != nil {
		t.Fatalf("RunWasmBuffer failed: %v", err)
	}
	if rets[0].(int32) != 144 {
		t.Errorf("fib(11) = %v, want 144", rets[0])
	}

	ast := loadASTFromFile(t, testWasmPath("fib.wasm"))
	defer ast.Release()
	rets, err = vm.RunWasmAST(ast, "fib", int32(12))
	if err != nil {
		t.Fatalf("RunWasmAST failed: %v", err)
	}
	if rets[0].(int32) != 233 {
		t.Errorf("fib(12) = %v, want 233", rets[0])
	}
}

func TestVMStepByStep(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	// Executing before instantiation is a workflow error.
	if _, err := vm.Execute("fib", int32(5)); err == nil {
		t.Error("Execute before instantiation should fail")
	}

	if err := vm.LoadWasmFile(testWasmPath("fib.wasm")); err != nil {
		t.Fatalf("LoadWasmFile failed: %v", err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatalf("Instantiate failed: %v", err)
	}
	rets, err := vm.Execute("fib", int32(13))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if rets[0].(int32) != 377 {
		t.Errorf("fib(13) = %v, want 377", rets[0])
	}

	// Unknown function.
	if _, err := vm.Execute("no_such_func"); err == nil {
		t.Error("executing an unknown function should fail")
	}

	// Function list and type queries.
	names, types := vm.GetFunctionList()
	if len(names) != 1 || names[0] != "fib" || len(types) != 1 {
		t.Errorf("GetFunctionList() = %v", names)
	}
	if vm.GetFunctionType("fib") == nil {
		t.Error("GetFunctionType(fib) should not be nil")
	}
	if vm.GetFunctionType("nope") != nil {
		t.Error("GetFunctionType(nope) should be nil")
	}
	if vm.GetActiveModule() == nil {
		t.Error("active module should exist after instantiation")
	}

	vm.Cleanup()
	if vm.GetActiveModule() != nil {
		t.Error("active module should be gone after Cleanup")
	}
}

func TestVMRegisteredModule(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	if err := vm.RegisterWasmFile("math", testWasmPath("fib.wasm")); err != nil {
		t.Fatalf("RegisterWasmFile failed: %v", err)
	}
	rets, err := vm.ExecuteRegistered("math", "fib", int32(10))
	if err != nil {
		t.Fatalf("ExecuteRegistered failed: %v", err)
	}
	if rets[0].(int32) != 89 {
		t.Errorf("math.fib(10) = %v, want 89", rets[0])
	}
	if vm.GetFunctionTypeRegistered("math", "fib") == nil {
		t.Error("GetFunctionTypeRegistered should find math.fib")
	}
	if vm.GetRegisteredModule("math") == nil {
		t.Error("GetRegisteredModule should find math")
	}

	buf, err := os.ReadFile(testWasmPath("memops.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	if err := vm.RegisterWasmBuffer("memops", buf); err != nil {
		t.Fatalf("RegisterWasmBuffer failed: %v", err)
	}

	ast := loadASTFromFile(t, testWasmPath("trap.wasm"))
	defer ast.Release()
	if err := vm.RegisterAST("trapmod", ast); err != nil {
		t.Fatalf("RegisterAST failed: %v", err)
	}

	// Plug-in provided modules may be auto-registered into the VM as well, so
	// only check that the explicitly registered modules are present.
	registered := map[string]bool{}
	for _, name := range vm.ListRegisteredModule() {
		registered[name] = true
	}
	for _, name := range []string{"math", "memops", "trapmod"} {
		if !registered[name] {
			t.Errorf("module %q missing from ListRegisteredModule()", name)
		}
	}

	if _, err := vm.ExecuteRegistered("trapmod", "trap"); err == nil {
		t.Error("registered trap function should fail")
	}
	if _, err := vm.ExecuteRegistered("ghost", "fib", int32(1)); err == nil {
		t.Error("executing in an unregistered module should fail")
	}
}

func TestVMRegisterHostModule(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	env := buildEnvModule(t, 12321)
	defer env.Release()
	if err := vm.RegisterModule(env); err != nil {
		t.Fatalf("RegisterModule failed: %v", err)
	}
	rets, err := vm.RunWasmFile(testWasmPath("types.wasm"), "get_hglob")
	if err != nil {
		t.Fatalf("get_hglob failed: %v", err)
	}
	if rets[0].(int32) != 12321 {
		t.Errorf("get_hglob = %v, want 12321", rets[0])
	}
}

func TestVMRegisterModuleWithAlias(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	env := buildEnvModule(t, 777)
	defer env.Release()

	// Register the module under an alias different from its own name.
	if err := vm.RegisterModuleWithAlias("aliased_env", env); err != nil {
		t.Fatalf("RegisterModuleWithAlias failed: %v", err)
	}
	if vm.GetRegisteredModule("aliased_env") == nil {
		t.Error("module should be registered under the alias")
	}
	rets, err := vm.ExecuteRegistered("aliased_env", "host_add", int32(2), int32(3))
	if err != nil {
		t.Fatalf("aliased host_add failed: %v", err)
	}
	if rets[0].(int32) != 5 {
		t.Errorf("aliased host_add = %v, want 5", rets[0])
	}
}

func TestVMForceDeleteRegisteredModule(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	// ForceDeleteRegisteredModule destroys the module instance, so the Go
	// wrapper must not be released afterwards.
	doomed := NewModule("doomed")
	gtype := NewGlobalType(NewValTypeI32(), ValMut_Const)
	doomed.AddGlobal("g", NewGlobal(gtype, int32(1)))
	gtype.Release()
	if err := vm.RegisterModule(doomed); err != nil {
		t.Fatalf("RegisterModule failed: %v", err)
	}
	if vm.GetRegisteredModule("doomed") == nil {
		t.Fatal("module should be registered")
	}

	vm.ForceDeleteRegisteredModule("doomed")
	if vm.GetRegisteredModule("doomed") != nil {
		t.Error("module should be gone after ForceDeleteRegisteredModule")
	}
	// Deleting a nonexistent module must not crash.
	vm.ForceDeleteRegisteredModule("no_such_module")
}

func TestVMWasi(t *testing.T) {
	conf := NewConfigure(WASI)
	defer conf.Release()
	vm := NewVMWithConfig(conf)
	defer vm.Release()

	wasi := vm.GetImportModule(WASI)
	if wasi == nil {
		t.Fatal("WASI import module not found")
	}
	wasi.InitWasi([]string{"hello.wasm"}, []string{}, []string{})

	if _, err := vm.RunWasmFile(testWasmPath("wasi_hello.wasm"), "_start"); err != nil {
		t.Fatalf("running the WASI module failed: %v", err)
	}
	if code := wasi.WasiGetExitCode(); code != 21 {
		t.Errorf("WASI exit code = %d, want 21", code)
	}
}

func TestVMWithStoreAndConfig(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()
	store := NewStore()
	defer store.Release()

	for _, vm := range []*VM{
		NewVMWithStore(store),
		NewVMWithConfigAndStore(conf, store),
	} {
		if vm == nil {
			t.Fatal("VM creation failed")
		}
		if vm.GetStore() == nil || vm.GetLoader() == nil ||
			vm.GetValidator() == nil || vm.GetExecutor() == nil ||
			vm.GetStatistics() == nil {
			t.Error("VM component getters should not return nil")
		}
		vm.Release()
	}
}

func TestVMAsync(t *testing.T) {
	vm := NewVM()
	defer vm.Release()

	async := vm.AsyncRunWasmFile(testWasmPath("fib.wasm"), "fib", int32(14))
	if async == nil {
		t.Fatal("AsyncRunWasmFile returned nil")
	}
	async.WaitFor(5000)
	rets, err := async.GetResult()
	if err != nil {
		t.Fatalf("async fib failed: %v", err)
	}
	if rets[0].(int32) != 610 {
		t.Errorf("fib(14) = %v, want 610", rets[0])
	}
	async.Release()

	// Load + async execute.
	if err := vm.LoadWasmBuffer(asyncLoopWasm); err != nil {
		t.Fatal(err)
	}
	if err := vm.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := vm.Instantiate(); err != nil {
		t.Fatal(err)
	}
	async = vm.AsyncExecute("_start")
	if async == nil {
		t.Fatal("AsyncExecute returned nil")
	}
	if async.WaitFor(1) {
		t.Error("infinite loop should not finish within 1ms")
	}
	async.Cancel()
	if _, err := async.GetResult(); err == nil {
		t.Error("canceled execution should report an error")
	} else if !strings.Contains(err.Error(), "interrupt") {
		t.Errorf("canceled execution error = %q, want an interruption error", err.Error())
	}
	async.Release()
}
