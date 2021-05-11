package wasmedge

// #include <wasmedge.h>
import "C"

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

type Configure struct {
	_inner *C.WasmEdge_ConfigureContext
}

func NewConfigure(params ...interface{}) *Configure {
	self := &Configure{
		_inner: C.WasmEdge_ConfigureCreate(),
	}

	if self._inner == nil {
		return nil
	}

	for _, conf := range params {
		switch conf.(type) {
		case Proposal:
			C.WasmEdge_ConfigureAddProposal(self._inner, C.enum_WasmEdge_Proposal(conf.(Proposal)))
		case HostRegistration:
			C.WasmEdge_ConfigureAddHostRegistration(self._inner, C.enum_WasmEdge_HostRegistration(conf.(HostRegistration)))
		default:
			panic("Wrong argument of NewConfigure()")
		}
	}

	return self
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

func (self *Configure) Delete() {
	C.WasmEdge_ConfigureDelete(self._inner)
	self._inner = nil
}
