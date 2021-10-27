package wasmedge

// #include <wasmedge/wasmedge.h>
// typedef uint32_t (*wasmedgego_GetExport)
//	 (const WasmEdge_StoreContext *, WasmEdge_String *, const uint32_t);
// typedef uint32_t (*wasmedgego_GetRegExport)
//	 (const WasmEdge_StoreContext *, WasmEdge_String, WasmEdge_String *, const uint32_t);
//
// uint32_t wasmedgego_WrapListExport(wasmedgego_GetExport f,
//						  const WasmEdge_StoreContext *Cxt,
//						  WasmEdge_String *Names,
//					      const uint32_t Len) {
//   return f(Cxt, Names, Len);
// }
// uint32_t wasmedgego_WrapListRegExport(wasmedgego_GetRegExport f,
//						  const WasmEdge_StoreContext *Cxt,
//						  WasmEdge_String ModName,
//						  WasmEdge_String *Names,
//					      const uint32_t Len) {
//   return f(Cxt, ModName, Names, Len);
// }
import "C"

type Store struct {
	_inner *C.WasmEdge_StoreContext
}

func (self *Store) getExports(exportlen C.uint32_t, getfunc interface{}, modname string) []string {
	cnames := make([]C.WasmEdge_String, int(exportlen))
	if int(exportlen) > 0 {
		switch getfunc.(type) {
		case C.wasmedgego_GetExport:
			C.wasmedgego_WrapListExport(getfunc.(C.wasmedgego_GetExport), self._inner, &cnames[0], exportlen)
		case C.wasmedgego_GetRegExport:
			cmodname := toWasmEdgeStringWrap(modname)
			C.wasmedgego_WrapListRegExport(getfunc.(C.wasmedgego_GetRegExport), self._inner, cmodname, &cnames[0], exportlen)
		}
	}
	names := make([]string, int(exportlen))
	for i := 0; i < int(exportlen); i++ {
		names[i] = fromWasmEdgeString(cnames[i])
		C.WasmEdge_StringDelete(cnames[i])
	}
	return names
}

func NewStore() *Store {
	self := &Store{
		_inner: C.WasmEdge_StoreCreate(),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *Store) FindFunction(name string) *Function {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindFunction(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Function{_inner: cinst}
}

func (self *Store) FindFunctionRegistered(modulename string, name string) *Function {
	cmodname := toWasmEdgeStringWrap(modulename)
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindFunctionRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Function{_inner: cinst}
}

func (self *Store) FindTable(name string) *Table {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindTable(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Table{_inner: cinst}
}

func (self *Store) FindTableRegistered(modulename string, name string) *Table {
	cmodname := toWasmEdgeStringWrap(modulename)
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindTableRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Table{_inner: cinst}
}

func (self *Store) FindMemory(name string) *Memory {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindMemory(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Memory{_inner: cinst}
}

func (self *Store) FindMemoryRegistered(modulename string, name string) *Memory {
	cmodname := toWasmEdgeStringWrap(modulename)
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindMemoryRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Memory{_inner: cinst}
}

func (self *Store) FindGlobal(name string) *Global {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindGlobal(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Global{_inner: cinst}
}

func (self *Store) FindGlobalRegistered(modulename string, name string) *Global {
	cmodname := toWasmEdgeStringWrap(modulename)
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_StoreFindGlobalRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Global{_inner: cinst}
}

func (self *Store) ListFunction() []string {
	return self.getExports(
		C.WasmEdge_StoreListFunctionLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_StoreListFunction),
		"")
}

func (self *Store) ListFunctionRegistered(modulename string) []string {
	cmodname := toWasmEdgeStringWrap(modulename)
	return self.getExports(
		C.WasmEdge_StoreListFunctionRegisteredLength(self._inner, cmodname),
		C.wasmedgego_GetRegExport(C.WasmEdge_StoreListFunctionRegistered),
		modulename)
}

func (self *Store) ListTable() []string {
	return self.getExports(
		C.WasmEdge_StoreListTableLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_StoreListTable),
		"")
}

func (self *Store) ListTableRegistered(modulename string) []string {
	cmodname := toWasmEdgeStringWrap(modulename)
	return self.getExports(
		C.WasmEdge_StoreListTableRegisteredLength(self._inner, cmodname),
		C.wasmedgego_GetRegExport(C.WasmEdge_StoreListTableRegistered),
		modulename)
}

func (self *Store) ListMemory() []string {
	return self.getExports(
		C.WasmEdge_StoreListMemoryLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_StoreListMemory),
		"")
}

func (self *Store) ListMemoryRegistered(modulename string) []string {
	cmodname := toWasmEdgeStringWrap(modulename)
	return self.getExports(
		C.WasmEdge_StoreListMemoryRegisteredLength(self._inner, cmodname),
		C.wasmedgego_GetRegExport(C.WasmEdge_StoreListMemoryRegistered),
		modulename)
}

func (self *Store) ListGlobal() []string {
	return self.getExports(
		C.WasmEdge_StoreListGlobalLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_StoreListGlobal),
		"")
}

func (self *Store) ListGlobalRegistered(modulename string) []string {
	cmodname := toWasmEdgeStringWrap(modulename)
	return self.getExports(
		C.WasmEdge_StoreListGlobalRegisteredLength(self._inner, cmodname),
		C.wasmedgego_GetRegExport(C.WasmEdge_StoreListGlobalRegistered),
		modulename)
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
		C.WasmEdge_StringDelete(cnames[i])
	}
	return names
}

func (self *Store) Delete() {
	C.WasmEdge_StoreDelete(self._inner)
	self._inner = nil
}
