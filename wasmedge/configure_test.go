package wasmedge

import "testing"

func TestConfigureProposals(t *testing.T) {
	conf := NewConfigure()
	if conf == nil {
		t.Fatal("NewConfigure() returned nil")
	}
	defer conf.Release()

	// The default configuration is the WASM 3.0 proposal set; threads is one
	// of the few proposals disabled by default.
	if conf.HasConfig(THREADS) {
		t.Error("threads should be disabled by default")
	}
	if !conf.HasConfig(MULTI_MEMORIES) || !conf.HasConfig(GC) {
		t.Error("the WASM 3.0 proposals should be enabled by default")
	}
	conf.AddConfig(THREADS)
	if !conf.HasConfig(THREADS) {
		t.Error("threads should be enabled after AddConfig")
	}
	conf.RemoveConfig(THREADS)
	if conf.HasConfig(THREADS) {
		t.Error("threads should be disabled after RemoveConfig")
	}

	// All proposal constants should be addable and removable.
	for _, prop := range []Proposal{
		IMPORT_EXPORT_MUT_GLOBALS,
		NON_TRAP_FLOAT_TO_INT_CONVERSIONS,
		SIGN_EXTENSION_OPERATORS,
		MULTI_VALUE,
		BULK_MEMORY_OPERATIONS,
		REFERENCE_TYPES,
		SIMD,
		TAIL_CALL,
		EXTENDED_CONST,
		FUNCTION_REFERENCES,
		GC,
		MULTI_MEMORIES,
		THREADS,
		RELAXED_SIMD,
		ANNOTATIONS,
		MEMORY64,
		EXCEPTION_HANDLING,
	} {
		conf.AddConfig(prop)
		if !conf.HasConfig(prop) {
			t.Errorf("proposal %d should be enabled after AddConfig", prop)
		}
	}
}

func TestConfigureHostRegistration(t *testing.T) {
	conf := NewConfigure(WASI)
	if conf == nil {
		t.Fatal("NewConfigure(WASI) returned nil")
	}
	defer conf.Release()
	if !conf.HasConfig(WASI) {
		t.Error("WASI should be registered")
	}
	conf.RemoveConfig(WASI)
	if conf.HasConfig(WASI) {
		t.Error("WASI should be removed")
	}
}

func TestConfigureWASMStandard(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()

	conf.SetWASMStandard(Standard_WASM_1)
	if conf.HasConfig(SIMD) {
		t.Error("WASM 1.0 should not include the SIMD proposal")
	}

	conf.SetWASMStandard(Standard_WASM_2)
	if !conf.HasConfig(SIMD) {
		t.Error("WASM 2.0 should include the SIMD proposal")
	}
	if conf.HasConfig(TAIL_CALL) {
		t.Error("WASM 2.0 should not include the tail-call proposal")
	}

	conf.SetWASMStandard(Standard_WASM_3)
	if !conf.HasConfig(TAIL_CALL) {
		t.Error("WASM 3.0 should include the tail-call proposal")
	}
	if !conf.HasConfig(GC) {
		t.Error("WASM 3.0 should include the GC proposal")
	}
}

func TestConfigureRunMode(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()

	if conf.GetRunMode() != RunMode_Interpreter {
		t.Errorf("default run mode should be the interpreter, got %d", conf.GetRunMode())
	}
	for _, mode := range []RunMode{RunMode_JIT, RunMode_AOT, RunMode_Interpreter} {
		conf.SetRunMode(mode)
		if conf.GetRunMode() != mode {
			t.Errorf("run mode should be %d, got %d", mode, conf.GetRunMode())
		}
	}
}

func TestConfigureMaxMemoryPage(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()

	conf.SetMaxMemoryPage(1234)
	if conf.GetMaxMemoryPage() != 1234 {
		t.Errorf("max memory page should be 1234, got %d", conf.GetMaxMemoryPage())
	}
	// The page count is 64-bit in WasmEdge 0.17.1.
	const bigPages = uint(1) << 40
	conf.SetMaxMemoryPage(bigPages)
	if conf.GetMaxMemoryPage() != bigPages {
		t.Errorf("max memory page should be %d, got %d", bigPages, conf.GetMaxMemoryPage())
	}
}

func TestConfigureCompilerOptions(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()

	for _, level := range []CompilerOptimizationLevel{
		CompilerOptLevel_O0, CompilerOptLevel_O1, CompilerOptLevel_O2,
		CompilerOptLevel_O3, CompilerOptLevel_Os, CompilerOptLevel_Oz,
	} {
		conf.SetCompilerOptimizationLevel(level)
		if conf.GetCompilerOptimizationLevel() != level {
			t.Errorf("optimization level should be %d, got %d", level, conf.GetCompilerOptimizationLevel())
		}
	}

	for _, format := range []CompilerOutputFormat{
		CompilerOutputFormat_Native, CompilerOutputFormat_Wasm,
	} {
		conf.SetCompilerOutputFormat(format)
		if conf.GetCompilerOutputFormat() != format {
			t.Errorf("output format should be %d, got %d", format, conf.GetCompilerOutputFormat())
		}
	}

	conf.SetCompilerDumpIR(true)
	if !conf.IsCompilerDumpIR() {
		t.Error("dump IR should be enabled")
	}
	conf.SetCompilerDumpIR(false)

	conf.SetCompilerGenericBinary(true)
	if !conf.IsCompilerGenericBinary() {
		t.Error("generic binary should be enabled")
	}
}

func TestConfigureStatisticsOptions(t *testing.T) {
	conf := NewConfigure()
	defer conf.Release()

	conf.SetStatisticsInstructionCounting(true)
	if !conf.IsStatisticsInstructionCounting() {
		t.Error("instruction counting should be enabled")
	}
	conf.SetStatisticsTimeMeasuring(true)
	if !conf.IsStatisticsTimeMeasuring() {
		t.Error("time measuring should be enabled")
	}
	conf.SetStatisticsCostMeasuring(true)
	if !conf.IsStatisticsCostMeasuring() {
		t.Error("cost measuring should be enabled")
	}
}
