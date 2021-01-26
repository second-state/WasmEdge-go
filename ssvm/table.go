package ssvm

// #include <ssvm.h>
import "C"

type Table struct {
	_inner *C.SSVM_TableInstanceContext
}
