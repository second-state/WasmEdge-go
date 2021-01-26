package ssvm

// #include <ssvm.h>
import "C"

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

func (self *Loader) Delete() {
	C.SSVM_LoaderDelete(self._inner)
	self._inner = nil
}
