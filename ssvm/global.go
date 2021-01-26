package ssvm

// #include <ssvm.h>
import "C"

type Global struct {
	_inner *C.SSVM_GlobalInstanceContext
}
