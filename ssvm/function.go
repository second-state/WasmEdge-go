package ssvm

// #include <ssvm.h>
import "C"

type Function struct {
	_inner *C.SSVM_FunctionInstanceContext
}
