package ssvm

// #include <ssvm.h>
import "C"

type Memory struct {
	_inner *C.SSVM_MemoryInstanceContext
}
