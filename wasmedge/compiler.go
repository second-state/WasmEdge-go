package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"runtime"
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
	res := &Compiler{_inner: compiler, _own: true}
	runtime.SetFinalizer(res, (*Compiler).Release)
	return res
}

func NewCompilerWithConfig(conf *Configure) *Compiler {
	compiler := C.WasmEdge_CompilerCreate(conf._inner)
	if compiler == nil {
		return nil
	}
	res := &Compiler{_inner: compiler, _own: true}
	runtime.SetFinalizer(res, (*Compiler).Release)
	return res
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
	runtime.SetFinalizer(self, nil)
	self._inner = nil
	self._own = false
}
