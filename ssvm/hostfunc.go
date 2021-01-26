package ssvm

// #include <ssvm.h>
import "C"

type HostFunction struct {
	_inner *C.SSVM_HostFunctionContext
}
