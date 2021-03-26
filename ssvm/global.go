package ssvm

// #include <ssvm.h>
import "C"

type Global struct {
	_inner *C.SSVM_GlobalInstanceContext
}

func NewGlobal(val interface{}, vtype ValMut) *Global {
	cval := toSSVMValue(val)
	self := &Global{
		_inner: C.SSVM_GlobalInstanceCreate(cval, C.enum_SSVM_Mutability(vtype)),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Global) GetValType() ValType {
	return ValType(C.SSVM_GlobalInstanceGetValType(self._inner))
}

func (self *Global) GetMutability() ValMut {
	return ValMut(C.SSVM_GlobalInstanceGetMutability(self._inner))
}

func (self *Global) GetValue() interface{} {
	cval := C.SSVM_GlobalInstanceGetValue(self._inner)
	return fromSSVMValue(cval, cval.Type)
}

func (self *Global) SetValue(val interface{}) {
	C.SSVM_GlobalInstanceSetValue(self._inner, toSSVMValue(val))
}

func (self *Global) Delete() {
	C.SSVM_GlobalInstanceDelete(self._inner)
	self._inner = nil
}
