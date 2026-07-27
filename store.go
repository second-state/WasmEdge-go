package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import "runtime"

type storeState struct {
	retained []*Module
	roots    []any
}

func (state *storeState) retain(m *Module) {
	if m == nil {
		return
	}
	state.retained = append(state.retained, m)
	if m.life.isOwned() {
		m.registrations = append(m.registrations, state)
	}
}

func (state *storeState) forget(modules []*Module) {
	if len(modules) == 0 {
		return
	}
	remove := make(map[*Module]struct{}, len(modules))
	for _, module := range modules {
		remove[module] = struct{}{}
	}
	keptModules := state.retained[:0]
	for _, module := range state.retained {
		if _, ok := remove[module]; !ok {
			keptModules = append(keptModules, module)
		}
	}
	clear(state.retained[len(keptModules):])
	state.retained = keptModules
}

// forgetLast removes one most-recent registration edge for every module in
// modules. VM teardown uses this to discard only registrations that VM added,
// preserving an earlier external-store registration of the same module.
func (state *storeState) forgetLast(modules []*Module) {
	for _, module := range modules {
		for i := len(state.retained) - 1; i >= 0; i-- {
			if state.retained[i] != module {
				continue
			}
			copy(state.retained[i:], state.retained[i+1:])
			last := len(state.retained) - 1
			state.retained[last] = nil
			state.retained = state.retained[:last]
			break
		}
	}
}

func (state *storeState) releaseAll() {
	state.retained = nil
	state.roots = nil
}

// Store is the registry of named module instances that instantiation links
// imports against. It does not own registered modules; it references them.
type Store struct {
	ptr         *C.WasmEdge_StoreContext
	life        lifetime
	state       *storeState
	moduleOwner func() any
}

// NewStore creates an empty store. WasmEdge treats allocation failure as an
// unrecoverable condition for this otherwise infallible constructor.
func NewStore() *Store {
	ptr := C.WasmEdge_StoreCreate()
	if ptr == nil {
		panic("wasmedge: native Store allocation failed")
	}
	s := &Store{ptr: ptr, state: &storeState{}}
	arm(s, &s.life, "Store", func() { C.WasmEdge_StoreDelete(ptr) })
	return s
}

func (s *Store) assertAlive() { s.life.assertAlive("Store") }

func (s *Store) acquireLease() (func(), error) {
	return s.life.acquire("Store")
}

func (s *Store) retainReference(root any) bool {
	if root == nil {
		return true
	}
	if s.life.isOwned() {
		s.state.roots = append(s.state.roots, root)
		return true
	}
	if owner, ok := s.life.owner.(referenceRetainer); ok {
		return owner.retainReference(root)
	}
	return false
}

// retain mirrors the C store's non-owning registration edge with a Go strong
// reference. WasmEdge_ModuleInstanceDelete automatically unregisters the
// module from every store; Module.Close notifies this state so the Go graph
// follows that native edge. Instantiated dependants take separate leases.
func (s *Store) retain(m *Module) {
	s.state.retain(m)
}

func (s *Store) forget(modules []*Module) {
	s.state.forget(modules)
}

func (s *Store) acquireDependencyLeases() ([]*Module, func(), error) {
	modules := make([]*Module, 0, len(s.state.retained))
	owners := make([]any, 0, len(s.state.retained))
	seen := make(map[*Module]struct{}, len(s.state.retained))
	for _, module := range s.state.retained {
		if module == nil {
			continue
		}
		owners = append(owners, module)
		if _, exists := seen[module]; exists {
			continue
		}
		seen[module] = struct{}{}
		modules = append(modules, module)
	}
	release, err := acquireLeases(owners...)
	if err != nil {
		return nil, nil, err
	}
	return modules, release, nil
}

// Module looks up a registered module by name (borrowed; owned by whoever
// registered it). The second result is false when the name is unknown.
func (s *Store) Module(name string) (*Module, bool) {
	s.assertAlive()
	defer runtime.KeepAlive(s)
	cname := newWEString(name)
	defer freeWEString(cname)
	ptr := C.WasmEdge_StoreFindModule(s.ptr, cname)
	var owner any = s
	if s.moduleOwner != nil {
		owner = s.moduleOwner()
	}
	for _, retained := range s.state.retained {
		if retained != nil && retained.ptr == ptr {
			owner = retained
			break
		}
	}
	m := borrowedModule(ptr, owner)
	return m, m != nil
}

// ModuleNames lists the registered module names.
func (s *Store) ModuleNames() []string {
	s.assertAlive()
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
	return s.life.close(func() {
		C.WasmEdge_StoreDelete(ptr)
		s.state.releaseAll()
	})
}
