package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

// Config declares engine settings for VM, Loader, Validator, Executor and
// Compiler constructors. It is plain data with no C lifetime: constructors
// materialize a WasmEdge_ConfigureContext from it, hand it to the engine
// (which copies the settings), and free it before returning. nil is always
// a valid *Config and means engine defaults.
//
// Zero values mean "engine default" throughout, so a partially filled
// literal composes naturally:
//
//	wasmedge.NewVM(&wasmedge.Config{WASI: true, RunMode: wasmedge.RunModeJIT})
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
	// WASI pre-registers the built-in WASI host module on a VM.
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
// defaults (O3, native output, everything off).
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

// Standard is a WASM standard version. The zero value keeps the engine
// default (the Go constants are offset by one from the C enum so that a
// zero Config field is distinguishable from an explicit WASM 1.0).
type Standard int32

const (
	StandardUnset Standard = 0
	StandardWASM1 Standard = C.WasmEdge_Standard_WASM_1 + 1
	StandardWASM2 Standard = C.WasmEdge_Standard_WASM_2 + 1
	StandardWASM3 Standard = C.WasmEdge_Standard_WASM_3 + 1
)

// Proposal is one WASM feature proposal.
type Proposal uint32

const (
	ProposalImportExportMutGlobals        Proposal = C.WasmEdge_Proposal_ImportExportMutGlobals
	ProposalNonTrapFloatToIntConversions  Proposal = C.WasmEdge_Proposal_NonTrapFloatToIntConversions
	ProposalSignExtensionOperators        Proposal = C.WasmEdge_Proposal_SignExtensionOperators
	ProposalMultiValue                    Proposal = C.WasmEdge_Proposal_MultiValue
	ProposalBulkMemoryOperations          Proposal = C.WasmEdge_Proposal_BulkMemoryOperations
	ProposalReferenceTypes                Proposal = C.WasmEdge_Proposal_ReferenceTypes
	ProposalSIMD                          Proposal = C.WasmEdge_Proposal_SIMD
	ProposalTailCall                      Proposal = C.WasmEdge_Proposal_TailCall
	ProposalExtendedConst                 Proposal = C.WasmEdge_Proposal_ExtendedConst
	ProposalFunctionReferences            Proposal = C.WasmEdge_Proposal_FunctionReferences
	ProposalGC                            Proposal = C.WasmEdge_Proposal_GC
	ProposalMultiMemories                 Proposal = C.WasmEdge_Proposal_MultiMemories
	ProposalRelaxSIMD                     Proposal = C.WasmEdge_Proposal_RelaxSIMD
	ProposalAnnotations                   Proposal = C.WasmEdge_Proposal_Annotations
	ProposalExceptionHandling             Proposal = C.WasmEdge_Proposal_ExceptionHandling
	ProposalMemory64                      Proposal = C.WasmEdge_Proposal_Memory64
	ProposalThreads                       Proposal = C.WasmEdge_Proposal_Threads
	ProposalComponent                     Proposal = C.WasmEdge_Proposal_Component
)

// RunMode selects the execution engine. The zero value (interpreter) is the
// engine default.
type RunMode uint32

const (
	RunModeInterpreter RunMode = C.WasmEdge_RunMode_Interpreter
	RunModeJIT         RunMode = C.WasmEdge_RunMode_JIT
	RunModeAOT         RunMode = C.WasmEdge_RunMode_AOT
	RunModeLazyJIT     RunMode = C.WasmEdge_RunMode_LazyJIT
)

// OptimizationLevel is an AOT/JIT optimization level. The zero value keeps
// the engine default (O3); explicit levels are offset by one from the C
// enum for that reason.
type OptimizationLevel int32

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
// keeps the engine default (native shared library); explicit formats are
// offset by one from the C enum.
type CompilerOutputFormat int32

const (
	OutputFormatUnset  CompilerOutputFormat = 0
	OutputFormatNative CompilerOutputFormat = C.WasmEdge_CompilerOutputFormat_Native + 1
	OutputFormatWasm   CompilerOutputFormat = C.WasmEdge_CompilerOutputFormat_Wasm + 1
)

// TODO(intern-easy): A13 — add String() methods for Proposal, Standard,
// RunMode, OptimizationLevel and CompilerOutputFormat in a new
// config_string.go, using the upstream names from enum.inc. Pattern:
// ValKind.String in valtype.go. Extend config_test.go with a table test.

// build materializes the C configure context; callers free it via the
// returned function immediately after the consuming constructor returns.
// A nil *Config yields a nil context, which every C constructor accepts as
// "defaults".
func (c *Config) build() (*C.WasmEdge_ConfigureContext, func()) {
	if c == nil {
		return nil, func() {}
	}
	cxt := C.WasmEdge_ConfigureCreate()
	free := func() {
		if cxt != nil {
			C.WasmEdge_ConfigureDelete(cxt)
		}
	}
	if cxt == nil {
		return nil, free
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
	return cxt, free
}
