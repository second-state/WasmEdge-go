package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"runtime"
	"unsafe"
)

type Loader struct {
	_inner *C.WasmEdge_LoaderContext
	_own   bool
}

func NewLoader() *Loader {
	loader := C.WasmEdge_LoaderCreate(nil)
	if loader == nil {
		return nil
	}
	res := &Loader{_inner: loader, _own: true}
	runtime.SetFinalizer(res, (*Loader).Release)
	return res
}

func NewLoaderWithConfig(conf *Configure) *Loader {
	loader := C.WasmEdge_LoaderCreate(conf._inner)
	if loader == nil {
		return nil
	}
	res := &Loader{_inner: loader, _own: true}
	runtime.SetFinalizer(res, (*Loader).Release)
	return res
}

func (self *Loader) LoadFile(path string) (*AST, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var module *C.WasmEdge_ASTModuleContext = nil
	result := C.WasmEdge_LoaderParseFromFile(self._inner, &module, cpath)
	if !C.WasmEdge_ResultOK(result) {
		return nil, newError(result)
	}
	res := &AST{_inner: module, _own: true}
	runtime.SetFinalizer(res, (*AST).Release)
	return res, nil
}

func (self *Loader) LoadBuffer(buf []byte) (*AST, error) {
	var module *C.WasmEdge_ASTModuleContext = nil
	result := C.WasmEdge_LoaderParseFromBuffer(self._inner, &module, (*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	if !C.WasmEdge_ResultOK(result) {
		return nil, newError(result)
	}
	res := &AST{_inner: module, _own: true}
	runtime.SetFinalizer(res, (*AST).Release)
	return res, nil
}

func (self *Loader) Release() {
	if self._own {
		C.WasmEdge_LoaderDelete(self._inner)
	}
	runtime.SetFinalizer(self, nil)
	self._inner = nil
	self._own = false
}
