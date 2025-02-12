package wasmedge

// #include <wasmedge/wasmedge.h>
// #include <stdlib.h>
import "C"
import (
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
	return &Loader{_inner: loader, _own: true}
}

func NewLoaderWithConfig(conf *Configure) *Loader {
	loader := C.WasmEdge_LoaderCreate(conf._inner)
	if loader == nil {
		return nil
	}
	return &Loader{_inner: loader, _own: true}
}

func (self *Loader) LoadFile(path string) (*AST, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var module *C.WasmEdge_ASTModuleContext = nil
	result := C.WasmEdge_LoaderParseFromFile(self._inner, &module, cpath)
	if !C.WasmEdge_ResultOK(result) {
		return nil, newError(result)
	}
	return &AST{_inner: module, _own: true}, nil
}

func (self *Loader) LoadBuffer(buf []byte) (*AST, error) {
	var module *C.WasmEdge_ASTModuleContext = nil
	cbytes := C.WasmEdge_BytesWrap((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	result := C.WasmEdge_LoaderParseFromBytes(self._inner, &module, cbytes)
	if !C.WasmEdge_ResultOK(result) {
		return nil, newError(result)
	}
	return &AST{_inner: module, _own: true}, nil
}

func (self *Loader) Release() {
	if self._own {
		C.WasmEdge_LoaderDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
