package spectest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

type specMode uint8

const (
	modeInterpreter specMode = 1 << iota
	modeAOT
	modeJIT
	modeAll = modeInterpreter | modeAOT | modeJIT
)

func (m specMode) String() string {
	switch m {
	case modeInterpreter:
		return "interpreter"
	case modeAOT:
		return "AOT"
	case modeJIT:
		return "JIT"
	}
	return fmt.Sprintf("mode(%d)", m)
}

type folderConfig struct {
	name      string
	standard  wasmedge.WASMStandard
	proposals []wasmedge.Proposal
	modes     specMode
}

// The folder configurations, mirroring TestsuiteProposals in WasmEdge
// test/spec. The component-model folder is intentionally not supported.
var folderConfigs = []folderConfig{
	{name: "wasm-1.0", standard: wasmedge.Standard_WASM_1, modes: modeAll},
	{name: "wasm-2.0", standard: wasmedge.Standard_WASM_2, modes: modeAll},
	{name: "wasm-3.0", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	{name: "wasm-3.0-bulk-memory", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	// The exception-handling proposal is not implemented in AOT/JIT yet.
	{name: "wasm-3.0-exceptions", standard: wasmedge.Standard_WASM_3, modes: modeInterpreter},
	{name: "wasm-3.0-gc", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	{name: "wasm-3.0-memory64", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	{name: "wasm-3.0-multi-memory", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	{name: "wasm-3.0-relaxed-simd", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	{name: "wasm-3.0-simd", standard: wasmedge.Standard_WASM_3, modes: modeAll},
	{name: "threads", standard: wasmedge.Standard_WASM_2,
		proposals: []wasmedge.Proposal{wasmedge.THREADS}, modes: modeAll},
}

// specVM bundles the per-unit execution state: a VM with the spectest host
// module registered, and the AOT compiler when running in AOT mode.
type specVM struct {
	vm       *wasmedge.VM
	specMod  *wasmedge.Module
	compiler *wasmedge.Compiler
	tmpdir   string
}

func newSpecVM(t *testing.T, folder folderConfig, mode specMode) *specVM {
	t.Helper()
	conf := wasmedge.NewConfigure()
	defer conf.Release()
	conf.SetWASMStandard(folder.standard)
	for _, proposal := range folder.proposals {
		conf.AddConfig(proposal)
	}
	switch mode {
	case modeJIT:
		conf.SetRunMode(wasmedge.RunMode_JIT)
	case modeAOT:
		conf.SetRunMode(wasmedge.RunMode_AOT)
	}

	ctx := &specVM{}
	ctx.vm = wasmedge.NewVMWithConfig(conf)
	if ctx.vm == nil {
		t.Fatal("failed to create the VM")
	}
	ctx.specMod = buildSpecTestModule(t)
	if err := ctx.vm.RegisterModule(ctx.specMod); err != nil {
		t.Fatalf("failed to register the spectest module: %v", err)
	}

	if mode == modeAOT {
		conf.SetCompilerOutputFormat(wasmedge.CompilerOutputFormat_Native)
		conf.SetCompilerOptimizationLevel(wasmedge.CompilerOptLevel_O0)
		ctx.compiler = wasmedge.NewCompilerWithConfig(conf)
		if ctx.compiler == nil {
			t.Fatal("failed to create the compiler")
		}
		tmpdir, err := os.MkdirTemp("", "wasmedge-spectest-aot-")
		if err != nil {
			t.Fatal(err)
		}
		ctx.tmpdir = tmpdir
	}
	return ctx
}

func (c *specVM) release() {
	c.vm.Release()
	c.specMod.Release()
	if c.compiler != nil {
		c.compiler.Release()
	}
	if c.tmpdir != "" {
		os.RemoveAll(c.tmpdir)
	}
}

// prepare returns the file to load: in AOT mode the module is compiled to
// a native shared library first, otherwise the original file is used.
func (c *specVM) prepare(path string) (string, error) {
	if c.compiler == nil {
		return path, nil
	}
	out := filepath.Join(c.tmpdir,
		strings.TrimSuffix(filepath.Base(path), ".wasm")+nativeLibExtension())
	if err := c.compiler.Compile(path, out); err != nil {
		return "", err
	}
	return out, nil
}

// instantiate loads, validates, and instantiates an already prepared file as
// the active module of the VM.
func (c *specVM) instantiate(prepared string) error {
	if err := c.vm.LoadWasmFile(prepared); err != nil {
		return err
	}
	if err := c.vm.Validate(); err != nil {
		return err
	}
	return c.vm.Instantiate()
}

func (c *specVM) load(path string) error {
	prepared, err := c.prepare(path)
	if err != nil {
		return err
	}
	return c.vm.LoadWasmFile(prepared)
}

func (c *specVM) validate(path string) error {
	prepared, err := c.prepare(path)
	if err != nil {
		return err
	}
	if err := c.vm.LoadWasmFile(prepared); err != nil {
		return err
	}
	return c.vm.Validate()
}

func (c *specVM) instantiateFresh(path string) error {
	prepared, err := c.prepare(path)
	if err != nil {
		return err
	}
	return c.instantiate(prepared)
}

func nativeLibExtension() string {
	switch runtime.GOOS {
	case "darwin":
		return ".dylib"
	case "windows":
		return ".dll"
	}
	return ".so"
}

// buildSpecTestModule creates the "spectest" host module required by the
// suites, mirroring createSpecTestModule in WasmEdge test/api/hostfunc_c.c.
func buildSpecTestModule(t *testing.T) *wasmedge.Module {
	t.Helper()
	mod := wasmedge.NewModule("spectest")
	if mod == nil {
		t.Fatal("failed to create the spectest module")
	}

	noop := func(interface{}, *wasmedge.CallingFrame, []interface{}) ([]interface{}, wasmedge.Result) {
		return nil, wasmedge.Result_Success
	}
	addPrint := func(name string, params ...*wasmedge.ValType) {
		ftype := wasmedge.NewFunctionType(params, nil)
		defer ftype.Release()
		fn := wasmedge.NewFunction(ftype, noop, nil, 0)
		if fn == nil {
			t.Fatalf("failed to create the host function %q", name)
		}
		mod.AddFunction(name, fn)
	}
	addPrint("print")
	addPrint("print_i32", wasmedge.NewValTypeI32())
	addPrint("print_i64", wasmedge.NewValTypeI64())
	addPrint("print_f32", wasmedge.NewValTypeF32())
	addPrint("print_f64", wasmedge.NewValTypeF64())
	addPrint("print_i32_f32", wasmedge.NewValTypeI32(), wasmedge.NewValTypeF32())
	addPrint("print_f64_f64", wasmedge.NewValTypeF64(), wasmedge.NewValTypeF64())

	addTable := func(name string, lim *wasmedge.Limit) {
		defer lim.Release()
		ttype := wasmedge.NewTableType(wasmedge.NewValTypeFuncRef(), lim)
		defer ttype.Release()
		tab := wasmedge.NewTable(ttype)
		if tab == nil {
			t.Fatalf("failed to create the host table %q", name)
		}
		mod.AddTable(name, tab)
	}
	addTable("table", wasmedge.NewLimitWithMax(10, 20))
	addTable("table64", wasmedge.NewLimit64WithMax(10, 20))

	addMemory := func(name string, lim *wasmedge.Limit) {
		defer lim.Release()
		mtype := wasmedge.NewMemoryType(lim)
		defer mtype.Release()
		mem := wasmedge.NewMemory(mtype)
		if mem == nil {
			t.Fatalf("failed to create the host memory %q", name)
		}
		mod.AddMemory(name, mem)
	}
	addMemory("memory", wasmedge.NewLimitWithMax(1, 2))
	addMemory("shared_memory", wasmedge.NewLimitSharedWithMax(1, 2))

	addGlobal := func(name string, vtype *wasmedge.ValType, value interface{}) {
		gtype := wasmedge.NewGlobalType(vtype, wasmedge.ValMut_Const)
		defer gtype.Release()
		glob := wasmedge.NewGlobal(gtype, value)
		if glob == nil {
			t.Fatalf("failed to create the host global %q", name)
		}
		mod.AddGlobal(name, glob)
	}
	addGlobal("global_i32", wasmedge.NewValTypeI32(), int32(666))
	addGlobal("global_i64", wasmedge.NewValTypeI64(), int64(666))
	addGlobal("global_f32", wasmedge.NewValTypeF32(), float32(666.6))
	addGlobal("global_f64", wasmedge.NewValTypeF64(), float64(666.6))

	return mod
}

// aotSupported probes whether the WasmEdge library was built with the AOT
// compiler.
func aotSupported(t *testing.T) bool {
	t.Helper()
	compiler := wasmedge.NewCompiler()
	if compiler == nil {
		return false
	}
	defer compiler.Release()
	out := filepath.Join(t.TempDir(), "probe"+nativeLibExtension())
	// An empty module: WASM magic and version only.
	err := compiler.CompileBuffer(
		[]byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}, out)
	return err == nil
}

func runSpecMode(t *testing.T, mode specMode) {
	root := suiteRoot(t)
	for _, folder := range folderConfigs {
		if folder.modes&mode == 0 {
			continue
		}
		for _, unit := range listUnits(t, root, folder.name) {
			folder, unit := folder, unit
			t.Run(folder.name+"/"+unit, func(t *testing.T) {
				runUnit(t, root, folder, mode, unit)
			})
		}
	}
}

func TestSpecInterpreter(t *testing.T) {
	runSpecMode(t, modeInterpreter)
}

func TestSpecAOT(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the AOT spec tests in short mode")
	}
	if !aotSupported(t) {
		t.Skip("the WasmEdge library was built without the AOT compiler")
	}
	runSpecMode(t, modeAOT)
}

func TestSpecJIT(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping the JIT spec tests in short mode")
	}
	if !aotSupported(t) {
		t.Skip("the WasmEdge library was built without the JIT compiler")
	}
	runSpecMode(t, modeJIT)
}

func TestMain(m *testing.M) {
	// The WasmEdge AOT/JIT trap signal handlers lack SA_ONSTACK, which the Go
	// runtime rejects; re-run the tests with async preemption disabled.
	if !strings.Contains(os.Getenv("GODEBUG"), "asyncpreemptoff=1") {
		godebug := "asyncpreemptoff=1"
		if prev := os.Getenv("GODEBUG"); prev != "" {
			godebug = prev + ",asyncpreemptoff=1"
		}
		cmd := exec.Command(os.Args[0], os.Args[1:]...)
		cmd.Env = append(os.Environ(), "GODEBUG="+godebug)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintln(os.Stderr, "failed to re-execute the test binary:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	wasmedge.SetLogOff()
	os.Exit(m.Run())
}
