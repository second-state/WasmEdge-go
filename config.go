package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "fmt"

// Config declares engine settings for VM, Loader, Validator, Executor and
// Compiler constructors. It is plain data with no C lifetime: constructors
// materialize a WasmEdge_ConfigureContext from it, hand it to the engine
// (which copies the settings), and free it before returning. nil is always
// a valid *Config and means engine defaults.
//
// Zero values mean "engine default" throughout, so a partially filled
// literal composes naturally:
//
//	wasmedge.NewVM(&wasmedge.Config{RunMode: wasmedge.RunModeJIT})
type Config struct {
	// Standard selects a WASM standard version, overriding the default
	// proposal set. StandardUnset keeps the engine default.
	Standard Standard
	// EnableProposals turns individual proposals on (applied after
	// Standard).
	EnableProposals []Proposal
	// DisableProposals turns individual proposals off (applied after
	// Standard and EnableProposals).
	DisableProposals []Proposal
	// WASI pre-registers the built-in WASI host module on a VM. Registration
	// alone grants no stdio, arguments, environment, or preopens. Call
	// VM.WASIModule and Module.InitWASI with an explicit WASIStdio policy
	// before exposing those capabilities.
	WASI bool
	// RunMode selects the execution engine. The zero value is the
	// interpreter, which is also the engine default.
	RunMode RunMode
	// MaxMemoryPages caps memory instances, in 64KiB pages. 0 keeps the
	// engine default.
	MaxMemoryPages uint64
	// AllowAFUNIX enables AF_UNIX support in WASI sockets.
	AllowAFUNIX bool
	// Compiler holds AOT/JIT compiler options.
	Compiler CompilerConfig
	// Stats holds statistics collection options.
	Stats StatsConfig
}

// CompilerConfig holds AOT/JIT compiler options. Zero values keep engine
// defaults (O3, universal Wasm output, everything off).
type CompilerConfig struct {
	OptimizationLevel OptimizationLevel
	OutputFormat      CompilerOutputFormat
	DumpIR            bool
	GenericBinary     bool
	Interruptible     bool
}

// StatsConfig holds statistics collection options; all default off.
type StatsConfig struct {
	InstructionCounting bool
	CostMeasuring       bool
	TimeMeasuring       bool
}

// EffectiveConfig is the configuration that WasmEdge will actually use after
// applying engine defaults and Config overrides. Unlike Config, every enum
// field is explicit and Proposals contains the complete enabled set.
//
// Obtain a snapshot with Config.Effective. The returned value is ordinary Go
// data and owns no native resource.
type EffectiveConfig struct {
	Proposals      []Proposal
	WASI           bool
	RunMode        RunMode
	MaxMemoryPages uint64
	AllowAFUNIX    bool
	Compiler       CompilerConfig
	Stats          StatsConfig
}

// Standard is a WASM standard version. The zero value keeps the engine
// default (the Go constants are offset by one from the C enum so that a
// zero Config field is distinguishable from an explicit WASM 1.0).
type Standard int32

// Standard values select the core WebAssembly language revision.
const (
	StandardUnset Standard = 0
	StandardWASM1 Standard = C.WasmEdge_Standard_WASM_1 + 1
	StandardWASM2 Standard = C.WasmEdge_Standard_WASM_2 + 1
	StandardWASM3 Standard = C.WasmEdge_Standard_WASM_3 + 1
)

// Proposal is one WASM feature proposal.
type Proposal uint32

// Proposal values identify WasmEdge feature toggles.
const (
	ProposalImportExportMutGlobals       Proposal = C.WasmEdge_Proposal_ImportExportMutGlobals
	ProposalNonTrapFloatToIntConversions Proposal = C.WasmEdge_Proposal_NonTrapFloatToIntConversions
	ProposalSignExtensionOperators       Proposal = C.WasmEdge_Proposal_SignExtensionOperators
	ProposalMultiValue                   Proposal = C.WasmEdge_Proposal_MultiValue
	ProposalBulkMemoryOperations         Proposal = C.WasmEdge_Proposal_BulkMemoryOperations
	ProposalReferenceTypes               Proposal = C.WasmEdge_Proposal_ReferenceTypes
	ProposalSIMD                         Proposal = C.WasmEdge_Proposal_SIMD
	ProposalTailCall                     Proposal = C.WasmEdge_Proposal_TailCall
	ProposalExtendedConst                Proposal = C.WasmEdge_Proposal_ExtendedConst
	ProposalFunctionReferences           Proposal = C.WasmEdge_Proposal_FunctionReferences
	ProposalGC                           Proposal = C.WasmEdge_Proposal_GC
	ProposalMultiMemories                Proposal = C.WasmEdge_Proposal_MultiMemories
	ProposalRelaxSIMD                    Proposal = C.WasmEdge_Proposal_RelaxSIMD
	ProposalAnnotations                  Proposal = C.WasmEdge_Proposal_Annotations
	ProposalExceptionHandling            Proposal = C.WasmEdge_Proposal_ExceptionHandling
	ProposalMemory64                     Proposal = C.WasmEdge_Proposal_Memory64
	ProposalThreads                      Proposal = C.WasmEdge_Proposal_Threads
	ProposalComponent                    Proposal = C.WasmEdge_Proposal_Component
)

// HostRegistration identifies a built-in host module that WasmEdge can
// register into a VM.
type HostRegistration uint32

// HostRegistration values identify built-in host modules.
const (
	HostRegistrationWASI HostRegistration = C.WasmEdge_HostRegistration_Wasi
)

// RunMode selects the execution engine. The zero value (interpreter) is the
// engine default.
type RunMode uint32

// RunMode values select the interpreter, JIT, or AOT execution path.
const (
	RunModeInterpreter RunMode = C.WasmEdge_RunMode_Interpreter
	RunModeJIT         RunMode = C.WasmEdge_RunMode_JIT
	RunModeAOT         RunMode = C.WasmEdge_RunMode_AOT
)

// OptimizationLevel is an AOT/JIT optimization level. The zero value keeps
// the engine default (O3); explicit levels are offset by one from the C
// enum for that reason.
type OptimizationLevel int32

// OptimizationLevel values select the compiler optimization policy.
const (
	OptimizationUnset OptimizationLevel = 0
	OptimizationO0    OptimizationLevel = C.WasmEdge_CompilerOptimizationLevel_O0 + 1
	OptimizationO1    OptimizationLevel = C.WasmEdge_CompilerOptimizationLevel_O1 + 1
	OptimizationO2    OptimizationLevel = C.WasmEdge_CompilerOptimizationLevel_O2 + 1
	OptimizationO3    OptimizationLevel = C.WasmEdge_CompilerOptimizationLevel_O3 + 1
	OptimizationOs    OptimizationLevel = C.WasmEdge_CompilerOptimizationLevel_Os + 1
	OptimizationOz    OptimizationLevel = C.WasmEdge_CompilerOptimizationLevel_Oz + 1
)

// CompilerOutputFormat selects the AOT artifact format. The zero value
// keeps the engine default (universal Wasm); explicit formats are
// offset by one from the C enum.
type CompilerOutputFormat int32

// CompilerOutputFormat values select native or universal Wasm artifacts.
const (
	OutputFormatUnset  CompilerOutputFormat = 0
	OutputFormatNative CompilerOutputFormat = C.WasmEdge_CompilerOutputFormat_Native + 1
	OutputFormatWasm   CompilerOutputFormat = C.WasmEdge_CompilerOutputFormat_Wasm + 1
)

// build validates the declarative configuration and materializes its C
// counterpart. Callers free the context via the returned function immediately
// after the consuming constructor returns. A nil *Config yields a nil context,
// which every C constructor accepts as "defaults".
func (c *Config) build() (*C.WasmEdge_ConfigureContext, func(), error) {
	if c == nil {
		return nil, func() {}, nil
	}
	if err := c.validate(); err != nil {
		return nil, func() {}, err
	}
	cxt := C.WasmEdge_ConfigureCreate()
	free := func() {
		if cxt != nil {
			C.WasmEdge_ConfigureDelete(cxt)
		}
	}
	if cxt == nil {
		return nil, free, fmt.Errorf("create native configuration: %w", ErrUnavailable)
	}
	if c.Standard != StandardUnset {
		C.WasmEdge_ConfigureSetWASMStandard(cxt, C.enum_WasmEdge_Standard(c.Standard-1))
	}
	for _, p := range c.EnableProposals {
		C.WasmEdge_ConfigureAddProposal(cxt, C.enum_WasmEdge_Proposal(p))
	}
	for _, p := range c.DisableProposals {
		C.WasmEdge_ConfigureRemoveProposal(cxt, C.enum_WasmEdge_Proposal(p))
	}
	if c.WASI {
		C.WasmEdge_ConfigureAddHostRegistration(cxt, C.WasmEdge_HostRegistration_Wasi)
	}
	if c.RunMode != RunModeInterpreter {
		C.WasmEdge_ConfigureSetRunMode(cxt, C.enum_WasmEdge_RunMode(c.RunMode))
	}
	if c.MaxMemoryPages > 0 {
		C.WasmEdge_ConfigureSetMaxMemoryPage(cxt, C.uint64_t(c.MaxMemoryPages))
	}
	if c.AllowAFUNIX {
		C.WasmEdge_ConfigureSetAllowAFUNIX(cxt, true)
	}
	if c.Compiler.OptimizationLevel != OptimizationUnset {
		C.WasmEdge_ConfigureCompilerSetOptimizationLevel(cxt,
			C.enum_WasmEdge_CompilerOptimizationLevel(c.Compiler.OptimizationLevel-1))
	}
	if c.Compiler.OutputFormat != OutputFormatUnset {
		C.WasmEdge_ConfigureCompilerSetOutputFormat(cxt,
			C.enum_WasmEdge_CompilerOutputFormat(c.Compiler.OutputFormat-1))
	}
	if c.Compiler.DumpIR {
		C.WasmEdge_ConfigureCompilerSetDumpIR(cxt, true)
	}
	if c.Compiler.GenericBinary {
		C.WasmEdge_ConfigureCompilerSetGenericBinary(cxt, true)
	}
	if c.Compiler.Interruptible {
		C.WasmEdge_ConfigureCompilerSetInterruptible(cxt, true)
	}
	if c.Stats.InstructionCounting {
		C.WasmEdge_ConfigureStatisticsSetInstructionCounting(cxt, true)
	}
	if c.Stats.CostMeasuring {
		C.WasmEdge_ConfigureStatisticsSetCostMeasuring(cxt, true)
	}
	if c.Stats.TimeMeasuring {
		C.WasmEdge_ConfigureStatisticsSetTimeMeasuring(cxt, true)
	}
	return cxt, free, nil
}

// Effective resolves engine defaults and Config overrides into a readable
// snapshot. A nil receiver is valid and reports the WasmEdge defaults.
func (c *Config) Effective() (EffectiveConfig, error) {
	source := c
	if source == nil {
		source = &Config{}
	}
	cxt, free, err := source.build()
	if err != nil {
		return EffectiveConfig{}, err
	}
	defer free()

	out := EffectiveConfig{
		WASI: bool(C.WasmEdge_ConfigureHasHostRegistration(
			cxt, C.WasmEdge_HostRegistration_Wasi)),
		RunMode:        RunMode(C.WasmEdge_ConfigureGetRunMode(cxt)),
		MaxMemoryPages: uint64(C.WasmEdge_ConfigureGetMaxMemoryPage(cxt)),
		AllowAFUNIX:    bool(C.WasmEdge_ConfigureIsAllowAFUNIX(cxt)),
		Compiler: CompilerConfig{
			OptimizationLevel: OptimizationLevel(
				C.WasmEdge_ConfigureCompilerGetOptimizationLevel(cxt),
			) + 1,
			OutputFormat: CompilerOutputFormat(
				C.WasmEdge_ConfigureCompilerGetOutputFormat(cxt),
			) + 1,
			DumpIR: bool(C.WasmEdge_ConfigureCompilerIsDumpIR(cxt)),
			GenericBinary: bool(
				C.WasmEdge_ConfigureCompilerIsGenericBinary(cxt),
			),
			Interruptible: bool(
				C.WasmEdge_ConfigureCompilerIsInterruptible(cxt),
			),
		},
		Stats: StatsConfig{
			InstructionCounting: bool(
				C.WasmEdge_ConfigureStatisticsIsInstructionCounting(cxt),
			),
			CostMeasuring: bool(
				C.WasmEdge_ConfigureStatisticsIsCostMeasuring(cxt),
			),
			TimeMeasuring: bool(
				C.WasmEdge_ConfigureStatisticsIsTimeMeasuring(cxt),
			),
		},
	}
	for _, proposal := range allProposals {
		if bool(C.WasmEdge_ConfigureHasProposal(
			cxt, C.enum_WasmEdge_Proposal(proposal),
		)) {
			out.Proposals = append(out.Proposals, proposal)
		}
	}
	return out, nil
}

var allProposals = [...]Proposal{
	ProposalImportExportMutGlobals,
	ProposalNonTrapFloatToIntConversions,
	ProposalSignExtensionOperators,
	ProposalMultiValue,
	ProposalBulkMemoryOperations,
	ProposalReferenceTypes,
	ProposalSIMD,
	ProposalTailCall,
	ProposalExtendedConst,
	ProposalFunctionReferences,
	ProposalGC,
	ProposalMultiMemories,
	ProposalRelaxSIMD,
	ProposalAnnotations,
	ProposalExceptionHandling,
	ProposalMemory64,
	ProposalThreads,
	ProposalComponent,
}

func (c *Config) validate() error {
	switch c.Standard {
	case StandardUnset, StandardWASM1, StandardWASM2, StandardWASM3:
	default:
		return fmt.Errorf("standard %s: %w", c.Standard, ErrInvalidArgument)
	}
	for i, p := range c.EnableProposals {
		if !p.valid() {
			return fmt.Errorf("enable proposal %d (%s): %w", i, p, ErrInvalidArgument)
		}
	}
	for i, p := range c.DisableProposals {
		if !p.valid() {
			return fmt.Errorf("disable proposal %d (%s): %w", i, p, ErrInvalidArgument)
		}
	}
	switch c.RunMode {
	case RunModeInterpreter, RunModeJIT, RunModeAOT:
	default:
		return fmt.Errorf("run mode %s: %w", c.RunMode, ErrInvalidArgument)
	}
	switch c.Compiler.OptimizationLevel {
	case OptimizationUnset, OptimizationO0, OptimizationO1, OptimizationO2,
		OptimizationO3, OptimizationOs, OptimizationOz:
	default:
		return fmt.Errorf("optimization level %s: %w",
			c.Compiler.OptimizationLevel, ErrInvalidArgument)
	}
	switch c.Compiler.OutputFormat {
	case OutputFormatUnset, OutputFormatNative, OutputFormatWasm:
	default:
		return fmt.Errorf("compiler output format %s: %w",
			c.Compiler.OutputFormat, ErrInvalidArgument)
	}
	return nil
}

func (p Proposal) valid() bool {
	for _, known := range allProposals {
		if p == known {
			return true
		}
	}
	return false
}
