package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type ImportObject struct {
	_inner     *C.WasmEdge_ImportObjectContext
	_hostfuncs []uint
}

func NewImportObject(modname string) *ImportObject {
	impobj := C.WasmEdge_ImportObjectCreate(toWasmEdgeStringWrap(modname))
	if impobj == nil {
		return nil
	}
	return &ImportObject{
		_inner: impobj,
	}
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
		_inner: C.WasmEdge_ImportObjectCreateWASI(ptrargs, C.uint32_t(len(cargs)),
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

	C.WasmEdge_ImportObjectInitWASI(self._inner,
		ptrargs, C.uint32_t(len(cargs)),
		ptrenvs, C.uint32_t(len(cenvs)),
		ptrdirs, C.uint32_t(len(cdirs)),
		ptrpreopens, C.uint32_t(len(cpreopens)))

	freeCStringArray(cargs)
	freeCStringArray(cenvs)
	freeCStringArray(cdirs)
	freeCStringArray(cpreopens)
}

func NewWasmEdgeProcessImportObject(allowedcmds []string, allowall bool) *ImportObject {
	ccmds := toCStringArray(allowedcmds)
	var ptrcmds *(*C.char) = nil
	if len(ccmds) > 0 {
		ptrcmds = &ccmds[0]
	}

	self := &ImportObject{
		_inner: C.
			WasmEdge_ImportObjectCreateWasmEdgeProcess(ptrcmds, C.uint32_t(len(ccmds)), C.bool(allowall)),
	}

	freeCStringArray(ccmds)
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *ImportObject) InitWasmEdgeProcess(allowedcmds []string, allowall bool) {
	ccmds := toCStringArray(allowedcmds)
	var ptrcmds *(*C.char) = nil
	if len(ccmds) > 0 {
		ptrcmds = &ccmds[0]
	}

	C.WasmEdge_ImportObjectInitWasmEdgeProcess(self._inner, ptrcmds, C.uint32_t(len(ccmds)), C.bool(allowall))

	freeCStringArray(ccmds)
}

func (self *ImportObject) AddFunction(name string, inst *Function) {
	hostfuncMgr.mu.Lock()
	defer hostfuncMgr.mu.Unlock()

	C.WasmEdge_ImportObjectAddFunction(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	self._hostfuncs = append(self._hostfuncs, inst._index)
	inst._inner = nil
}

func (self *ImportObject) AddTable(name string, inst *Table) {
	C.WasmEdge_ImportObjectAddTable(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	inst._inner = nil
}

func (self *ImportObject) AddMemory(name string, inst *Memory) {
	C.WasmEdge_ImportObjectAddMemory(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	inst._inner = nil
}

func (self *ImportObject) AddGlobal(name string, inst *Global) {
	C.WasmEdge_ImportObjectAddGlobal(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	inst._inner = nil
}

func (self *ImportObject) Delete() {
	for _, idx := range self._hostfuncs {
		hostfuncMgr.del(idx)
	}
	self._hostfuncs = []uint{}
	C.WasmEdge_ImportObjectDelete(self._inner)
	self._inner = nil
}
