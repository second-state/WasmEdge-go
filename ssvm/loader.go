package ssvm

// #include <ssvm.h>
import "C"
import "unsafe"

type Loader struct {
	_inner *C.SSVM_LoaderContext
}

func NewLoader() *Loader {
	self := &Loader{
		_inner: C.SSVM_LoaderCreate(nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewLoaderWithConfig(conf *Configure) *Loader {
	self := &Loader{
		_inner: C.SSVM_LoaderCreate(conf._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Loader) LoadFile(path string) (*AST, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var module *C.SSVM_ASTModuleContext = nil
	res := C.SSVM_LoaderParseFromFile(self._inner, &module, cpath)
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return &AST{_inner: module}, nil
}

func (self *Loader) LoadBuffer(buf []byte) (*AST, error) {
	var module *C.SSVM_ASTModuleContext = nil
	res := C.SSVM_LoaderParseFromBuffer(self._inner, &module, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf)))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return &AST{_inner: module}, nil
}

func (self *Loader) Delete() {
	C.SSVM_LoaderDelete(self._inner)
	self._inner = nil
}
