package ssvm

// #include <ssvm.h>
import "C"

type ImportObject struct {
	_inner *C.SSVM_ImportObjectContext
}

func NewImportObject(modname string) *ImportObject {
	self := &ImportObject{
		_inner: C.SSVM_ImportObjectCreate(toSSVMStringWrap(modname)),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewWasiImportObject(args []string, envs []string, dirs []string, preopens []string) *ImportObject {
	cargs := toCStringArray(args)
	cenvs := toCStringArray(envs)
	cdirs := toCStringArray(dirs)
	cpreopens := toCStringArray(preopens)
	var ptrargs *(*C.char) = nil
	var ptrenvs *(*C.char) = nil
	var ptrdirs *(*C.char) = nil
	var ptrpreopens *(*C.char) = nil
	if len(cargs) > 0 {
		ptrargs = &cargs[0]
	}
	if len(cenvs) > 0 {
		ptrenvs = &cenvs[0]
	}
	if len(cdirs) > 0 {
		ptrdirs = &cdirs[0]
	}
	if len(cpreopens) > 0 {
		ptrpreopens = &cpreopens[0]
	}

	self := &ImportObject{
		_inner: C.SSVM_ImportObjectCreateWASI(ptrargs, C.uint32_t(len(cargs)),
			ptrenvs, C.uint32_t(len(cenvs)),
			ptrdirs, C.uint32_t(len(cdirs)),
			ptrpreopens, C.uint32_t(len(cpreopens))),
	}

	freeCStringArray(cargs)
	freeCStringArray(cenvs)
	freeCStringArray(cdirs)
	freeCStringArray(cpreopens)

	if self._inner == nil {
		return nil
	}
	return self
}

func (self *ImportObject) InitWasi(args []string, envs []string, dirs []string, preopens []string) {
	cargs := toCStringArray(args)
	cenvs := toCStringArray(envs)
	cdirs := toCStringArray(dirs)
	cpreopens := toCStringArray(preopens)
	var ptrargs *(*C.char) = nil
	var ptrenvs *(*C.char) = nil
	var ptrdirs *(*C.char) = nil
	var ptrpreopens *(*C.char) = nil
	if len(cargs) > 0 {
		ptrargs = &cargs[0]
	}
	if len(cenvs) > 0 {
		ptrenvs = &cenvs[0]
	}
	if len(cdirs) > 0 {
		ptrdirs = &cdirs[0]
	}
	if len(cpreopens) > 0 {
		ptrpreopens = &cpreopens[0]
	}

	C.SSVM_ImportObjectInitWASI(self._inner,
		ptrargs, C.uint32_t(len(cargs)),
		ptrenvs, C.uint32_t(len(cenvs)),
		ptrdirs, C.uint32_t(len(cdirs)),
		ptrpreopens, C.uint32_t(len(cpreopens)))

	freeCStringArray(cargs)
	freeCStringArray(cenvs)
	freeCStringArray(cdirs)
	freeCStringArray(cpreopens)
}

func (self *ImportObject) AddHostFunction(name string, inst *HostFunction) {
	C.SSVM_ImportObjectAddHostFunction(self._inner, toSSVMStringWrap(name), inst._inner)
}

func (self *ImportObject) AddTable(name string, inst *Table) {
	C.SSVM_ImportObjectAddTable(self._inner, toSSVMStringWrap(name), inst._inner)
}

func (self *ImportObject) AddMemory(name string, inst *Memory) {
	C.SSVM_ImportObjectAddMemory(self._inner, toSSVMStringWrap(name), inst._inner)
}

func (self *ImportObject) AddGlobal(name string, inst *Global) {
	C.SSVM_ImportObjectAddGlobal(self._inner, toSSVMStringWrap(name), inst._inner)
}

func (self *ImportObject) Delete() {
	C.SSVM_ImportObjectDelete(self._inner)
	self._inner = nil
}
