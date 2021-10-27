package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import "unsafe"

type Compiler struct {
	_inner *C.WasmEdge_CompilerContext
}

func NewCompiler() *Compiler {
	self := &Compiler{
		_inner: C.WasmEdge_CompilerCreate(nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewCompilerWithConfig(conf *Configure) *Compiler {
	self := &Compiler{
		_inner: C.WasmEdge_CompilerCreate(conf._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
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

func (self *Compiler) Delete() {
	C.WasmEdge_CompilerDelete(self._inner)
	self._inner = nil
}
