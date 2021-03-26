package ssvm

// #include <ssvm.h>
// typedef uint32_t (*ssvmgo_GetExport)
//	 (const SSVM_StoreContext *, SSVM_String *, const uint32_t);
// typedef uint32_t (*ssvmgo_GetRegExport)
//	 (const SSVM_StoreContext *, SSVM_String, SSVM_String *, const uint32_t);
//
// uint32_t ssvmgo_WrapListExport(ssvmgo_GetExport f,
//						  const SSVM_StoreContext *Cxt,
//						  SSVM_String *Names,
//					      const uint32_t Len) {
//   return f(Cxt, Names, Len);
// }
// uint32_t ssvmgo_WrapListRegExport(ssvmgo_GetRegExport f,
//						  const SSVM_StoreContext *Cxt,
//						  SSVM_String ModName,
//						  SSVM_String *Names,
//					      const uint32_t Len) {
//   return f(Cxt, ModName, Names, Len);
// }
import "C"

type Store struct {
	_inner *C.SSVM_StoreContext
}

func (self *Store) getExports(exportlen C.uint32_t, getfunc interface{}, modname string) []string {
	cnames := make([]C.SSVM_String, int(exportlen))
	if int(exportlen) > 0 {
		switch getfunc.(type) {
		case C.ssvmgo_GetExport:
			C.ssvmgo_WrapListExport(getfunc.(C.ssvmgo_GetExport), self._inner, &cnames[0], exportlen)
		case C.ssvmgo_GetRegExport:
			cmodname := toSSVMStringWrap(modname)
			C.ssvmgo_WrapListRegExport(getfunc.(C.ssvmgo_GetRegExport), self._inner, cmodname, &cnames[0], exportlen)
		}
	}
	names := make([]string, int(exportlen))
	for i := 0; i < int(exportlen); i++ {
		names[i] = fromSSVMString(cnames[i])
		C.SSVM_StringDelete(cnames[i])
	}
	return names
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

func (self *Store) FindFunction(name string) *Function {
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindFunction(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Function{_inner: cinst}
}

func (self *Store) FindFunctionRegistered(modulename string, name string) *Function {
	cmodname := toSSVMStringWrap(modulename)
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindFunctionRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Function{_inner: cinst}
}

func (self *Store) FindTable(name string) *Table {
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindTable(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Table{_inner: cinst}
}

func (self *Store) FindTableRegistered(modulename string, name string) *Table {
	cmodname := toSSVMStringWrap(modulename)
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindTableRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Table{_inner: cinst}
}

func (self *Store) FindMemory(name string) *Memory {
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindMemory(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Memory{_inner: cinst}
}

func (self *Store) FindMemoryRegistered(modulename string, name string) *Memory {
	cmodname := toSSVMStringWrap(modulename)
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindMemoryRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Memory{_inner: cinst}
}

func (self *Store) FindGlobal(name string) *Global {
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindGlobal(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Global{_inner: cinst}
}

func (self *Store) FindGlobalRegistered(modulename string, name string) *Global {
	cmodname := toSSVMStringWrap(modulename)
	cname := toSSVMStringWrap(name)
	cinst := C.SSVM_StoreFindGlobalRegistered(self._inner, cmodname, cname)
	if cinst == nil {
		return nil
	}
	return &Global{_inner: cinst}
}

func (self *Store) ListFunction() []string {
	return self.getExports(
		C.SSVM_StoreListFunctionLength(self._inner),
		C.ssvmgo_GetExport(C.SSVM_StoreListFunction),
		"")
}

func (self *Store) ListFunctionRegistered(modulename string) []string {
	cmodname := toSSVMStringWrap(modulename)
	return self.getExports(
		C.SSVM_StoreListFunctionRegisteredLength(self._inner, cmodname),
		C.ssvmgo_GetRegExport(C.SSVM_StoreListFunctionRegistered),
		modulename)
}

func (self *Store) ListTable() []string {
	return self.getExports(
		C.SSVM_StoreListTableLength(self._inner),
		C.ssvmgo_GetExport(C.SSVM_StoreListTable),
		"")
}

func (self *Store) ListTableRegistered(modulename string) []string {
	cmodname := toSSVMStringWrap(modulename)
	return self.getExports(
		C.SSVM_StoreListTableRegisteredLength(self._inner, cmodname),
		C.ssvmgo_GetRegExport(C.SSVM_StoreListTableRegistered),
		modulename)
}

func (self *Store) ListMemory() []string {
	return self.getExports(
		C.SSVM_StoreListMemoryLength(self._inner),
		C.ssvmgo_GetExport(C.SSVM_StoreListMemory),
		"")
}

func (self *Store) ListMemoryRegistered(modulename string) []string {
	cmodname := toSSVMStringWrap(modulename)
	return self.getExports(
		C.SSVM_StoreListMemoryRegisteredLength(self._inner, cmodname),
		C.ssvmgo_GetRegExport(C.SSVM_StoreListMemoryRegistered),
		modulename)
}

func (self *Store) ListGlobal() []string {
	return self.getExports(
		C.SSVM_StoreListGlobalLength(self._inner),
		C.ssvmgo_GetExport(C.SSVM_StoreListGlobal),
		"")
}

func (self *Store) ListGlobalRegistered(modulename string) []string {
	cmodname := toSSVMStringWrap(modulename)
	return self.getExports(
		C.SSVM_StoreListGlobalRegisteredLength(self._inner, cmodname),
		C.ssvmgo_GetRegExport(C.SSVM_StoreListGlobalRegistered),
		modulename)
}

func (self *Store) ListModule() []string {
	modlen := C.SSVM_StoreListModuleLength(self._inner)
	cnames := make([]C.SSVM_String, int(modlen))
	if int(modlen) > 0 {
		C.SSVM_StoreListModule(self._inner, &cnames[0], modlen)
	}
	names := make([]string, int(modlen))
	for i := 0; i < int(modlen); i++ {
		names[i] = fromSSVMString(cnames[i])
		C.SSVM_StringDelete(cnames[i])
	}
	return names
}

func (self *Store) Delete() {
	C.SSVM_StoreDelete(self._inner)
	self._inner = nil
}
