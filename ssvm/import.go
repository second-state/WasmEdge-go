package ssvm

// #include <ssvm.h>
import "C"

type ImportObject struct {
	_inner *C.SSVM_ImportObjectContext
}

func (self *ImportObject) Delete() {
	C.SSVM_ImportObjectDelete(self._inner)
	self._inner = nil
}
