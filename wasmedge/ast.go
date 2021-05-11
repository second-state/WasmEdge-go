package wasmedge

// #include <wasmedge.h>
import "C"

type AST struct {
	_inner *C.WasmEdge_ASTModuleContext
}

func (self *AST) Delete() {
	C.WasmEdge_ASTModuleDelete(self._inner)
	self._inner = nil
}
