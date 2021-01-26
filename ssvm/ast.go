package ssvm

// #include <ssvm.h>
import "C"

type AST struct {
	_inner *C.SSVM_ASTModuleContext
}

func (self *AST) Delete() {
	C.SSVM_ASTModuleDelete(self._inner)
	self._inner = nil
}
