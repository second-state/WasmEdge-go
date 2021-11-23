package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"unsafe"
)

type Compiler struct {
	_inner *C.WasmEdge_CompilerContext
	_own   bool
}

func NewCompiler() *Compiler {
	compiler := C.WasmEdge_CompilerCreate(nil)
	if compiler == nil {
		return nil
	}
	return &Compiler{_inner: compiler, _own: true}
}

func NewCompilerWithConfig(conf *Configure) *Compiler {
	compiler := C.WasmEdge_CompilerCreate(conf._inner)
	if compiler == nil {
		return nil
	}
	return &Compiler{_inner: compiler, _own: true}
}

func (self *Compiler) Compile(inpath string, outpath string) error {
	cinpath := C.CString(inpath)
	coutpath := C.CString(outpath)
	defer C.free(unsafe.Pointer(cinpath))
	defer C.free(unsafe.Pointer(coutpath))
	res := C.WasmEdge_CompilerCompile(self._inner, cinpath, coutpath)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Compiler) Release() {
	if self._own {
		C.WasmEdge_CompilerDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
