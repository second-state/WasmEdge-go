package wasmedge

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestConfigEnumString(t *testing.T) {
	tests := []struct {
		name string
		got  fmt.Stringer
		want string
	}{
		{"standard unset", StandardUnset, "unset"},
		{"standard wasm 1", StandardWASM1, "WASM_1"},
		{"standard wasm 2", StandardWASM2, "WASM_2"},
		{"standard wasm 3", StandardWASM3, "WASM_3"},
		{"standard unknown", Standard(-17), "Standard(-17)"},
		{
			"proposal mutable globals",
			ProposalImportExportMutGlobals,
			"ImportExportMutGlobals",
		},
		{
			"proposal non-trapping float conversion",
			ProposalNonTrapFloatToIntConversions,
			"NonTrapFloatToIntConversions",
		},
		{"proposal sign extension", ProposalSignExtensionOperators, "SignExtensionOperators"},
		{"proposal multi value", ProposalMultiValue, "MultiValue"},
		{"proposal bulk memory", ProposalBulkMemoryOperations, "BulkMemoryOperations"},
		{"proposal reference types", ProposalReferenceTypes, "ReferenceTypes"},
		{"proposal simd", ProposalSIMD, "SIMD"},
		{"proposal tail call", ProposalTailCall, "TailCall"},
		{"proposal extended const", ProposalExtendedConst, "ExtendedConst"},
		{"proposal function references", ProposalFunctionReferences, "FunctionReferences"},
		{"proposal gc", ProposalGC, "GC"},
		{"proposal multi memories", ProposalMultiMemories, "MultiMemories"},
		{"proposal relaxed simd", ProposalRelaxSIMD, "RelaxSIMD"},
		{"proposal annotations", ProposalAnnotations, "Annotations"},
		{"proposal exception handling", ProposalExceptionHandling, "ExceptionHandling"},
		{"proposal memory64", ProposalMemory64, "Memory64"},
		{"proposal threads", ProposalThreads, "Threads"},
		{"proposal component", ProposalComponent, "Component"},
		{"proposal unknown", Proposal(^uint32(0)), "Proposal(4294967295)"},
		{"run mode interpreter", RunModeInterpreter, "Interpreter"},
		{"run mode jit", RunModeJIT, "JIT"},
		{"run mode aot", RunModeAOT, "AOT"},
		{"run mode unknown", RunMode(99), "RunMode(99)"},
		{"optimization unset", OptimizationUnset, "unset"},
		{"optimization o0", OptimizationO0, "O0"},
		{"optimization o1", OptimizationO1, "O1"},
		{"optimization o2", OptimizationO2, "O2"},
		{"optimization o3", OptimizationO3, "O3"},
		{"optimization os", OptimizationOs, "Os"},
		{"optimization oz", OptimizationOz, "Oz"},
		{
			"optimization unknown",
			OptimizationLevel(-23),
			"OptimizationLevel(-23)",
		},
		{"output format unset", OutputFormatUnset, "unset"},
		{"output format native", OutputFormatNative, "Native"},
		{"output format wasm", OutputFormatWasm, "Wasm"},
		{
			"output format unknown",
			CompilerOutputFormat(-29),
			"CompilerOutputFormat(-29)",
		},
		{"host registration wasi", HostRegistrationWASI, "Wasi"},
		{
			"host registration unknown",
			HostRegistration(^uint32(0)),
			"HostRegistration(4294967295)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got.String(); got != tt.want {
				t.Fatalf("String(): got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigRejectsUnknownEnums(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"standard", Config{Standard: Standard(-1)}},
		{"enabled proposal", Config{EnableProposals: []Proposal{Proposal(^uint32(0))}}},
		{"disabled proposal", Config{DisableProposals: []Proposal{Proposal(^uint32(0))}}},
		{"run mode", Config{RunMode: RunMode(99)}},
		{
			"optimization level",
			Config{Compiler: CompilerConfig{OptimizationLevel: OptimizationLevel(-1)}},
		},
		{
			"compiler output format",
			Config{Compiler: CompilerConfig{OutputFormat: CompilerOutputFormat(-1)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, free, err := tt.cfg.build()
			defer free()
			if ctx != nil {
				t.Fatal("invalid configuration allocated a native context")
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("build error: got %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestConfigErrorPropagatesFromConstructor(t *testing.T) {
	invalid := &Config{RunMode: RunMode(99)}
	tests := []struct {
		name string
		run  func() error
	}{
		{"loader", func() error { _, err := NewLoader(invalid); return err }},
		{"validator", func() error { _, err := NewValidator(invalid); return err }},
		{"executor", func() error { _, err := NewExecutor(invalid); return err }},
		{"VM", func() error { _, err := NewVM(invalid); return err }},
		{"compiler", func() error { _, err := NewCompiler(invalid); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("constructor error: got %v, want ErrInvalidArgument", err)
			}
		})
	}
}

func TestConstructorsRejectNilOptions(t *testing.T) {
	if _, err := NewExecutor(nil, nil); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil ExecutorOption: got %v, want ErrInvalidArgument", err)
	}
	if _, err := NewVM(nil, nil); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("nil VMOption: got %v, want ErrInvalidArgument", err)
	}
}

func TestConfigEffectiveReportsNativeDefaults(t *testing.T) {
	fromNil, err := (*Config)(nil).Effective()
	if err != nil {
		t.Fatal(err)
	}
	fromZero, err := (&Config{}).Effective()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(fromNil, fromZero) {
		t.Fatalf("nil and zero Config differ:\nnil:  %#v\nzero: %#v", fromNil, fromZero)
	}
	if fromNil.Compiler.OptimizationLevel != OptimizationO3 {
		t.Fatalf(
			"default optimization: got %s, want O3",
			fromNil.Compiler.OptimizationLevel,
		)
	}
	if fromNil.Compiler.OutputFormat != OutputFormatWasm {
		t.Fatalf(
			"default output format: got %s, want Wasm",
			fromNil.Compiler.OutputFormat,
		)
	}
}

func TestConfigEffectiveReportsOverrides(t *testing.T) {
	cfg := &Config{
		Standard:         StandardWASM1,
		EnableProposals:  []Proposal{ProposalThreads},
		DisableProposals: []Proposal{ProposalMultiValue},
		WASI:             true,
		RunMode:          RunModeJIT,
		MaxMemoryPages:   123,
		AllowAFUNIX:      true,
		Compiler: CompilerConfig{
			OptimizationLevel: OptimizationO1,
			OutputFormat:      OutputFormatWasm,
			DumpIR:            true,
			GenericBinary:     true,
			Interruptible:     true,
		},
		Stats: StatsConfig{
			InstructionCounting: true,
			CostMeasuring:       true,
			TimeMeasuring:       true,
		},
	}
	got, err := cfg.Effective()
	if err != nil {
		t.Fatal(err)
	}
	if !got.WASI || got.RunMode != RunModeJIT || got.MaxMemoryPages != 123 ||
		!got.AllowAFUNIX {
		t.Fatalf("runtime overrides not preserved: %#v", got)
	}
	if got.Compiler != cfg.Compiler {
		t.Fatalf("compiler overrides: got %#v, want %#v", got.Compiler, cfg.Compiler)
	}
	if got.Stats != cfg.Stats {
		t.Fatalf("statistics overrides: got %#v, want %#v", got.Stats, cfg.Stats)
	}
	if !containsProposal(got.Proposals, ProposalThreads) {
		t.Fatalf("enabled proposal missing from %v", got.Proposals)
	}
	if containsProposal(got.Proposals, ProposalMultiValue) {
		t.Fatalf("disabled proposal present in %v", got.Proposals)
	}
}

func containsProposal(proposals []Proposal, want Proposal) bool {
	for _, proposal := range proposals {
		if proposal == want {
			return true
		}
	}
	return false
}
