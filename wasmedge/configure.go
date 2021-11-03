package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import "runtime"

type Proposal C.enum_WasmEdge_Proposal

const (
	ANNOTATIONS            = Proposal(C.WasmEdge_Proposal_Annotations)
	BULK_MEMORY_OPERATIONS = Proposal(C.WasmEdge_Proposal_BulkMemoryOperations)
	EXCEPTION_HANDLING     = Proposal(C.WasmEdge_Proposal_ExceptionHandling)
	FUNCTION_REFERENCES    = Proposal(C.WasmEdge_Proposal_FunctionReferences)
	MEMORY64               = Proposal(C.WasmEdge_Proposal_Memory64)
	REFERENCE_TYPES        = Proposal(C.WasmEdge_Proposal_ReferenceTypes)
	SIMD                   = Proposal(C.WasmEdge_Proposal_SIMD)
	TAIL_CALL              = Proposal(C.WasmEdge_Proposal_TailCall)
	THREADS                = Proposal(C.WasmEdge_Proposal_Threads)
)

type HostRegistration C.enum_WasmEdge_HostRegistration

const (
	WASI             = HostRegistration(C.WasmEdge_HostRegistration_Wasi)
	WasmEdge_PROCESS = HostRegistration(C.WasmEdge_HostRegistration_WasmEdge_Process)
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

	res := &Configure{_inner: conf, _own: true}
	runtime.SetFinalizer(res, (*Configure).Release)
	return res
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

func (self *Configure) SetCompilerOptimizationLevel(level CompilerOptimizationLevel) {
	C.WasmEdge_ConfigureCompilerSetOptimizationLevel(self._inner, C.enum_WasmEdge_CompilerOptimizationLevel(level))
}

func (self *Configure) GetCompilerOptimizationLevel() CompilerOptimizationLevel {
	return CompilerOptimizationLevel(C.WasmEdge_ConfigureCompilerGetOptimizationLevel(self._inner))
}

func (self *Configure) SetCompilerDumpIR(isdump bool) {
	C.WasmEdge_ConfigureCompilerSetDumpIR(self._inner, C.bool(isdump))
}

func (self *Configure) IsCompilerDumpIR() bool {
	return bool(C.WasmEdge_ConfigureCompilerIsDumpIR(self._inner))
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
	runtime.SetFinalizer(self, nil)
	self._inner = nil
	self._own = false
}
