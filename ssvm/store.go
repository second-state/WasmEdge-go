package ssvm

// #include <ssvm.h>
import "C"

type Store struct {
	_inner *C.SSVM_StoreContext
}

func NewStore() *Store {
	self := &Store{
		_inner: C.SSVM_StoreCreate(),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Store) Delete() {
	C.SSVM_StoreDelete(self._inner)
	self._inner = nil
}
