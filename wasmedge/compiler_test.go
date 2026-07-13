package wasmedge

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func nativeLibExtension() string {
	if runtime.GOOS == "darwin" {
		return ".dylib"
	}
	return ".so"
}

func TestCompilerUniversalWasm(t *testing.T) {
	compiler := NewCompiler()
	if compiler == nil {
		t.Fatal("NewCompiler() returned nil")
	}
	defer compiler.Release()

	out := filepath.Join(t.TempDir(), "fib_aot.wasm")
	if err := compiler.Compile(testWasmPath("fib.wasm"), out); err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	// The universal WASM output runs on a default VM.
	vm := NewVM()
	defer vm.Release()
	rets, err := vm.RunWasmFile(out, "fib", int32(20))
	if err != nil {
		t.Fatalf("running the compiled module failed: %v", err)
	}
	if rets[0].(int32) != 10946 {
		t.Errorf("fib(20) = %v, want 10946", rets[0])
	}
}

func TestCompilerBuffer(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()
	conf.SetCompilerOptimizationLevel(CompilerOptLevel_O0)
	compiler := NewCompilerWithConfig(conf)
	defer compiler.Release()

	buf, err := os.ReadFile(testWasmPath("fib.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "fib_buf_aot.wasm")
	if err := compiler.CompileBuffer(buf, out); err != nil {
		t.Fatalf("CompileBuffer failed: %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("compiled output missing: %v", err)
	}
}

func TestCompilerFailure(t *testing.T) {
	compiler := NewCompiler()
	defer compiler.Release()

	out := filepath.Join(t.TempDir(), "bad.wasm")
	if err := compiler.Compile("testdata/no_such_file.wasm", out); err == nil {
		t.Error("compiling a missing file should fail")
	}
	if err := compiler.CompileBuffer([]byte{0x00, 0x01}, out); err == nil {
		t.Error("compiling malformed bytes should fail")
	}
}

func TestRunModeAOT(t *testing.T) {
	// Compile to a native shared library and run it with the AOT run mode,
	// mirroring the WasmEdge AOT spec test flow.
	conf := NewConfigure()
	defer conf.Release()
	conf.SetCompilerOutputFormat(CompilerOutputFormat_Native)
	conf.SetCompilerOptimizationLevel(CompilerOptLevel_O0)
	compiler := NewCompilerWithConfig(conf)
	defer compiler.Release()

	out := filepath.Join(t.TempDir(), "fib"+nativeLibExtension())
	if err := compiler.Compile(testWasmPath("fib.wasm"), out); err != nil {
		t.Fatalf("native compile failed: %v", err)
	}

	runconf := NewConfigure()
	defer runconf.Release()
	runconf.SetRunMode(RunMode_AOT)
	vm := NewVMWithConfig(runconf)
	defer vm.Release()
	rets, err := vm.RunWasmFile(out, "fib", int32(18))
	if err != nil {
		t.Fatalf("running the native module failed: %v", err)
	}
	if rets[0].(int32) != 4181 {
		t.Errorf("fib(18) = %v, want 4181", rets[0])
	}
}

func TestRunModeJIT(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()
	conf.SetRunMode(RunMode_JIT)
	vm := NewVMWithConfig(conf)
	defer vm.Release()

	rets, err := vm.RunWasmFile(testWasmPath("fib.wasm"), "fib", int32(16))
	if err != nil {
		t.Fatalf("JIT run failed: %v", err)
	}
	if rets[0].(int32) != 1597 {
		t.Errorf("fib(16) = %v, want 1597", rets[0])
	}
}
