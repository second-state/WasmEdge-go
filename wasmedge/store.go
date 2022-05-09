package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type Store struct {
	_inner *C.WasmEdge_StoreContext
	_own   bool
}

func NewStore() *Store {
	store := C.WasmEdge_StoreCreate()
	if store == nil {
		return nil
	}
	return &Store{_inner: store, _own: true}
}

func (self *Store) FindModule(name string) *Module {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindModule(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Module{_inner: cinst, _own: false}
}

func (self *Store) ListModule() []string {
	modlen := C.WasmEdge_StoreListModuleLength(self._inner)
	cnames := make([]C.WasmEdge_String, int(modlen))
	if int(modlen) > 0 {
		C.WasmEdge_StoreListModule(self._inner, &cnames[0], modlen)
	}
	names := make([]string, int(modlen))
	for i := 0; i < int(modlen); i++ {
		names[i] = fromWasmEdgeString(cnames[i])
	}
	return names
}

func (self *Store) Release() {
	if self._own {
		C.WasmEdge_StoreDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
