package ssvm

// #include <ssvm.h>
import "C"

type Proposal C.enum_SSVM_Proposal

const (
	ANNOTATIONS            = Proposal(C.SSVM_Proposal_Annotations)
	BULK_MEMORY_OPERATIONS = Proposal(C.SSVM_Proposal_BulkMemoryOperations)
	EXCEPTION_HANDLING     = Proposal(C.SSVM_Proposal_ExceptionHandling)
	FUNCTION_REFERENCES    = Proposal(C.SSVM_Proposal_FunctionReferences)
	MEMORY64               = Proposal(C.SSVM_Proposal_Memory64)
	REFERENCE_TYPES        = Proposal(C.SSVM_Proposal_ReferenceTypes)
	SIMD                   = Proposal(C.SSVM_Proposal_SIMD)
	TAIL_CALL              = Proposal(C.SSVM_Proposal_TailCall)
	THREADS                = Proposal(C.SSVM_Proposal_Threads)
)

type HostRegistration C.enum_SSVM_HostRegistration

const (
	WASI         = HostRegistration(C.SSVM_HostRegistration_Wasi)
	SSVM_PROCESS = HostRegistration(C.SSVM_HostRegistration_SSVM_Process)
)

type Configure struct {
	_inner *C.SSVM_ConfigureContext
}

func NewConfigure(params ...interface{}) *Configure {
	self := &Configure{
		_inner: C.SSVM_ConfigureCreate(),
	}

	if self._inner == nil {
		return nil
	}

	for _, conf := range params {
		switch conf.(type) {
		case Proposal:
			C.SSVM_ConfigureAddProposal(self._inner, C.enum_SSVM_Proposal(conf.(Proposal)))
		case HostRegistration:
			C.SSVM_ConfigureAddHostRegistration(self._inner, C.enum_SSVM_HostRegistration(conf.(HostRegistration)))
		default:
			panic("Wrong argument of NewConfigure()")
		}
	}

	return self
}

func (self *Configure) HasConfig(conf interface{}) bool {
	switch conf.(type) {
	case Proposal:
		return bool(C.SSVM_ConfigureHasProposal(self._inner, C.enum_SSVM_Proposal(conf.(Proposal))))
	case HostRegistration:
		return bool(C.SSVM_ConfigureHasHostRegistration(self._inner, C.enum_SSVM_HostRegistration(conf.(HostRegistration))))
	default:
		panic("Wrong argument of Configure.HasConfig()")
	}
}

func (self *Configure) AddConfig(conf interface{}) {
	switch conf.(type) {
	case Proposal:
		C.SSVM_ConfigureAddProposal(self._inner, C.enum_SSVM_Proposal(conf.(Proposal)))
	case HostRegistration:
		C.SSVM_ConfigureAddHostRegistration(self._inner, C.enum_SSVM_HostRegistration(conf.(HostRegistration)))
	default:
		panic("Wrong argument of Configure.AddConfig()")
	}
}

func (self *Configure) RemoveConfig(conf interface{}) {
	switch conf.(type) {
	case Proposal:
		C.SSVM_ConfigureRemoveProposal(self._inner, C.enum_SSVM_Proposal(conf.(Proposal)))
	case HostRegistration:
		C.SSVM_ConfigureRemoveHostRegistration(self._inner, C.enum_SSVM_HostRegistration(conf.(HostRegistration)))
	default:
		panic("Wrong argument of Configure.RemoveConfig()")
	}
}

func (self *Configure) Delete() {
	C.SSVM_ConfigureDelete(self._inner)
	self._inner = nil
}
