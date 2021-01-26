package ssvm

// #include <ssvm.h>
import "C"

type Module struct {
	_inner *C.SSVM_ModuleInstanceContext
}
