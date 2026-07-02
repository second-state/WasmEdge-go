package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

// Store is the registry of named module instances that instantiation links
// imports against. It does not own registered modules; it references them.
type Store struct {
	ptr  *C.WasmEdge_StoreContext
	life lifetime
}

// NewStore creates an empty store.
func NewStore() *Store {
	ptr := C.WasmEdge_StoreCreate()
	if ptr == nil {
		return nil
	}
	s := &Store{ptr: ptr}
	arm(s, &s.life, "Store", func() { C.WasmEdge_StoreDelete(ptr) })
	return s
}

// Module looks up a registered module by name (borrowed; owned by whoever
// registered it). The second result is false when the name is unknown.
func (s *Store) Module(name string) (*Module, bool) {
	defer runtime.KeepAlive(s)
	cname := newWEString(name)
	defer freeWEString(cname)
	m := borrowedModule(C.WasmEdge_StoreFindModule(s.ptr, cname))
	return m, m != nil
}

// ModuleNames lists the registered module names.
func (s *Store) ModuleNames() []string {
	defer runtime.KeepAlive(s)
	return listStrings(
		func() C.uint32_t { return C.WasmEdge_StoreListModuleLength(s.ptr) },
		func(buf *C.WasmEdge_String, n C.uint32_t) C.uint32_t {
			return C.WasmEdge_StoreListModule(s.ptr, buf, n)
		})
}

// Close frees the store. Modules registered into it are not destroyed
// (they are owned by their creators), but they must not be used through
// this store afterwards. No-op after the first call.
func (s *Store) Close() error {
	ptr := s.ptr
	return s.life.close(func() { C.WasmEdge_StoreDelete(ptr) })
}
