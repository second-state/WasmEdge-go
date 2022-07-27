package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Proposal C.enum_WasmEdge_Proposal

const (
	IMPORT_EXPORT_MUT_GLOBALS         = Proposal(C.WasmEdge_Proposal_ImportExportMutGlobals)
	NON_TRAP_FLOAT_TO_INT_CONVERSIONS = Proposal(C.WasmEdge_Proposal_NonTrapFloatToIntConversions)
	SIGN_EXTENSION_OPERATORS          = Proposal(C.WasmEdge_Proposal_SignExtensionOperators)
	MULTI_VALUE                       = Proposal(C.WasmEdge_Proposal_MultiValue)
	BULK_MEMORY_OPERATIONS            = Proposal(C.WasmEdge_Proposal_BulkMemoryOperations)
	REFERENCE_TYPES                   = Proposal(C.WasmEdge_Proposal_ReferenceTypes)
	SIMD                              = Proposal(C.WasmEdge_Proposal_SIMD)
	TAIL_CALL                         = Proposal(C.WasmEdge_Proposal_TailCall)
	MULTI_MEMORIES                    = Proposal(C.WasmEdge_Proposal_MultiMemories)
	ANNOTATIONS                       = Proposal(C.WasmEdge_Proposal_Annotations)
	MEMORY64                          = Proposal(C.WasmEdge_Proposal_Memory64)
	EXCEPTION_HANDLING                = Proposal(C.WasmEdge_Proposal_ExceptionHandling)
	EXTENDED_CONST                    = Proposal(C.WasmEdge_Proposal_ExtendedConst)
	THREADS                           = Proposal(C.WasmEdge_Proposal_Threads)
	FUNCTION_REFERENCES               = Proposal(C.WasmEdge_Proposal_FunctionReferences)
)

type HostRegistration C.enum_WasmEdge_HostRegistration

const (
	WASI                        = HostRegistration(C.WasmEdge_HostRegistration_Wasi)
	WasmEdge_PROCESS            = HostRegistration(C.WasmEdge_HostRegistration_WasmEdge_Process)
	WasiNN                      = HostRegistration(C.WasmEdge_HostRegistration_WasiNN)
	WasiCrypto_Common           = HostRegistration(C.WasmEdge_HostRegistration_WasiCrypto_Common)
	WasiCrypto_AsymmetricCommon = HostRegistration(C.WasmEdge_HostRegistration_WasiCrypto_AsymmetricCommon)
	WasiCrypto_Kx               = HostRegistration(C.WasmEdge_HostRegistration_WasiCrypto_Kx)
	WasiCrypto_Signatures       = HostRegistration(C.WasmEdge_HostRegistration_WasiCrypto_Signatures)
	WasiCrypto_Symmetric        = HostRegistration(C.WasmEdge_HostRegistration_WasiCrypto_Symmetric)
)

type CompilerOptimizationLevel C.enum_WasmEdge_CompilerOptimizationLevel

const (
	// Disable as many optimizations as possible.
	CompilerOptLevel_O0 = CompilerOptimizationLevel(C.WasmEdge_CompilerOptimizationLevel_O0)
	// Optimize quickly without destroying debuggability.
	CompilerOptLevel_O1 = CompilerOptimizationLevel(C.WasmEdge_CompilerOptimizationLevel_O1)
	// Optimize for fast execution as much as possible without triggering significant incremental compile time or code size growth.
	CompilerOptLevel_O2 = CompilerOptimizationLevel(C.WasmEdge_CompilerOptimizationLevel_O2)
	// Optimize for fast execution as much as possible.
	CompilerOptLevel_O3 = CompilerOptimizationLevel(C.WasmEdge_CompilerOptimizationLevel_O3)
	// Optimize for small code size as much as possible without triggering significant incremental compile time or execution time slowdowns.
	CompilerOptLevel_Os = CompilerOptimizationLevel(C.WasmEdge_CompilerOptimizationLevel_Os)
	/// Optimize for small code size as much as possible.
	CompilerOptLevel_Oz = CompilerOptimizationLevel(C.WasmEdge_CompilerOptimizationLevel_Oz)
)

type CompilerOutputFormat C.enum_WasmEdge_CompilerOutputFormat

const (
	/// Native dynamic library format.
	CompilerOutputFormat_Native = CompilerOutputFormat(C.WasmEdge_CompilerOutputFormat_Native)
	/// WebAssembly with AOT compiled codes in custom section.
	CompilerOutputFormat_Wasm = CompilerOutputFormat(C.WasmEdge_CompilerOutputFormat_Wasm)
)

type Configure struct {
	_inner *C.WasmEdge_ConfigureContext
	_own   bool
}

func NewConfigure(params ...interface{}) *Configure {
	conf := C.WasmEdge_ConfigureCreate()
	if conf == nil {
		return nil
	}

	for _, val := range params {
		switch val.(type) {
		case Proposal:
			C.WasmEdge_ConfigureAddProposal(conf, C.enum_WasmEdge_Proposal(val.(Proposal)))
		case HostRegistration:
			C.WasmEdge_ConfigureAddHostRegistration(conf, C.enum_WasmEdge_HostRegistration(val.(HostRegistration)))
		default:
			panic("Wrong argument of NewConfigure()")
		}
	}

	return &Configure{_inner: conf, _own: true}
}

func (self *Configure) HasConfig(conf interface{}) bool {
	switch conf.(type) {
	case Proposal:
		return bool(C.WasmEdge_ConfigureHasProposal(self._inner, C.enum_WasmEdge_Proposal(conf.(Proposal))))
	case HostRegistration:
		return bool(C.WasmEdge_ConfigureHasHostRegistration(self._inner, C.enum_WasmEdge_HostRegistration(conf.(HostRegistration))))
	default:
		panic("Wrong argument of Configure.HasConfig()")
	}
}

func (self *Configure) AddConfig(conf interface{}) {
	switch conf.(type) {
	case Proposal:
		C.WasmEdge_ConfigureAddProposal(self._inner, C.enum_WasmEdge_Proposal(conf.(Proposal)))
	case HostRegistration:
		C.WasmEdge_ConfigureAddHostRegistration(self._inner, C.enum_WasmEdge_HostRegistration(conf.(HostRegistration)))
	default:
		panic("Wrong argument of Configure.AddConfig()")
	}
}

func (self *Configure) RemoveConfig(conf interface{}) {
	switch conf.(type) {
	case Proposal:
		C.WasmEdge_ConfigureRemoveProposal(self._inner, C.enum_WasmEdge_Proposal(conf.(Proposal)))
	case HostRegistration:
		C.WasmEdge_ConfigureRemoveHostRegistration(self._inner, C.enum_WasmEdge_HostRegistration(conf.(HostRegistration)))
	default:
		panic("Wrong argument of Configure.RemoveConfig()")
	}
}

func (self *Configure) SetMaxMemoryPage(pagesize uint) {
	C.WasmEdge_ConfigureSetMaxMemoryPage(self._inner, C.uint32_t(pagesize))
}

func (self *Configure) GetMaxMemoryPage() uint {
	return uint(C.WasmEdge_ConfigureGetMaxMemoryPage(self._inner))
}

func (self *Configure) SetCompilerOptimizationLevel(level CompilerOptimizationLevel) {
	C.WasmEdge_ConfigureCompilerSetOptimizationLevel(self._inner, C.enum_WasmEdge_CompilerOptimizationLevel(level))
}

func (self *Configure) GetCompilerOptimizationLevel() CompilerOptimizationLevel {
	return CompilerOptimizationLevel(C.WasmEdge_ConfigureCompilerGetOptimizationLevel(self._inner))
}

func (self *Configure) SetCompilerOutputFormat(format CompilerOutputFormat) {
	C.WasmEdge_ConfigureCompilerSetOutputFormat(self._inner, C.enum_WasmEdge_CompilerOutputFormat(format))
}

func (self *Configure) GetCompilerOutputFormat() CompilerOutputFormat {
	return CompilerOutputFormat(C.WasmEdge_ConfigureCompilerGetOutputFormat(self._inner))
}

func (self *Configure) SetCompilerDumpIR(isdump bool) {
	C.WasmEdge_ConfigureCompilerSetDumpIR(self._inner, C.bool(isdump))
}

func (self *Configure) IsCompilerDumpIR() bool {
	return bool(C.WasmEdge_ConfigureCompilerIsDumpIR(self._inner))
}

func (self *Configure) SetCompilerGenericBinary(isgeneric bool) {
	C.WasmEdge_ConfigureCompilerSetGenericBinary(self._inner, C.bool(isgeneric))
}

func (self *Configure) IsCompilerGenericBinary() bool {
	return bool(C.WasmEdge_ConfigureCompilerIsGenericBinary(self._inner))
}

func (self *Configure) SetStatisticsInstructionCounting(iscount bool) {
	C.WasmEdge_ConfigureStatisticsSetInstructionCounting(self._inner, C.bool(iscount))
}

func (self *Configure) IsStatisticsInstructionCounting() bool {
	return bool(C.WasmEdge_ConfigureStatisticsIsInstructionCounting(self._inner))
}

func (self *Configure) SetStatisticsTimeMeasuring(ismeasure bool) {
	C.WasmEdge_ConfigureStatisticsSetTimeMeasuring(self._inner, C.bool(ismeasure))
}

func (self *Configure) IsStatisticsTimeMeasuring() bool {
	return bool(C.WasmEdge_ConfigureStatisticsIsTimeMeasuring(self._inner))
}

func (self *Configure) SetStatisticsCostMeasuring(ismeasure bool) {
	C.WasmEdge_ConfigureStatisticsSetCostMeasuring(self._inner, C.bool(ismeasure))
}

func (self *Configure) IsStatisticsCostMeasuring() bool {
	return bool(C.WasmEdge_ConfigureStatisticsIsCostMeasuring(self._inner))
}

func (self *Configure) Release() {
	if self._own {
		C.WasmEdge_ConfigureDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
