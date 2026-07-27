package wasmedge

import "fmt"

// String returns the WasmEdge enum name for s.
func (s Standard) String() string {
	switch s {
	case StandardUnset:
		return "unset"
	case StandardWASM1:
		return "WASM_1"
	case StandardWASM2:
		return "WASM_2"
	case StandardWASM3:
		return "WASM_3"
	default:
		return fmt.Sprintf("Standard(%d)", int32(s))
	}
}

// String returns the WasmEdge enum name for p.
func (p Proposal) String() string {
	switch p {
	case ProposalImportExportMutGlobals:
		return "ImportExportMutGlobals"
	case ProposalNonTrapFloatToIntConversions:
		return "NonTrapFloatToIntConversions"
	case ProposalSignExtensionOperators:
		return "SignExtensionOperators"
	case ProposalMultiValue:
		return "MultiValue"
	case ProposalBulkMemoryOperations:
		return "BulkMemoryOperations"
	case ProposalReferenceTypes:
		return "ReferenceTypes"
	case ProposalSIMD:
		return "SIMD"
	case ProposalTailCall:
		return "TailCall"
	case ProposalExtendedConst:
		return "ExtendedConst"
	case ProposalFunctionReferences:
		return "FunctionReferences"
	case ProposalGC:
		return "GC"
	case ProposalMultiMemories:
		return "MultiMemories"
	case ProposalRelaxSIMD:
		return "RelaxSIMD"
	case ProposalAnnotations:
		return "Annotations"
	case ProposalExceptionHandling:
		return "ExceptionHandling"
	case ProposalMemory64:
		return "Memory64"
	case ProposalThreads:
		return "Threads"
	case ProposalComponent:
		return "Component"
	default:
		return fmt.Sprintf("Proposal(%d)", uint32(p))
	}
}

// String returns the WasmEdge enum name for m.
func (m RunMode) String() string {
	switch m {
	case RunModeInterpreter:
		return "Interpreter"
	case RunModeJIT:
		return "JIT"
	case RunModeAOT:
		return "AOT"
	default:
		return fmt.Sprintf("RunMode(%d)", uint32(m))
	}
}

// String returns the WasmEdge enum name for l.
func (l OptimizationLevel) String() string {
	switch l {
	case OptimizationUnset:
		return "unset"
	case OptimizationO0:
		return "O0"
	case OptimizationO1:
		return "O1"
	case OptimizationO2:
		return "O2"
	case OptimizationO3:
		return "O3"
	case OptimizationOs:
		return "Os"
	case OptimizationOz:
		return "Oz"
	default:
		return fmt.Sprintf("OptimizationLevel(%d)", int32(l))
	}
}

// String returns the WasmEdge enum name for f.
func (f CompilerOutputFormat) String() string {
	switch f {
	case OutputFormatUnset:
		return "unset"
	case OutputFormatNative:
		return "Native"
	case OutputFormatWasm:
		return "Wasm"
	default:
		return fmt.Sprintf("CompilerOutputFormat(%d)", int32(f))
	}
}

// String returns the WasmEdge enum name for h.
func (h HostRegistration) String() string {
	switch h {
	case HostRegistrationWASI:
		return "Wasi"
	default:
		return fmt.Sprintf("HostRegistration(%d)", uint32(h))
	}
}
