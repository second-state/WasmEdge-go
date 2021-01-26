package ssvm

/*
#include <ssvm.h>
size_t _GoStringLen(_GoString_ s);
const char *_GoStringPtr(_GoString_ s);
*/
import "C"

type VM struct {
	_inner *C.SSVM_VMContext
}

func NewVM() *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithConfig(conf *Configure) *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithStore(store *Store) *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(nil, store._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithConfigAndStore(conf *Configure, store *Store) *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(conf._inner, store._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *VM) Delete() {
	C.SSVM_VMDelete(self._inner)
	self._inner = nil
}
