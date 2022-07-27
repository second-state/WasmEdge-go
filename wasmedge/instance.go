package wasmedge

/*
#include <wasmedge/wasmedge.h>

typedef void (*wasmedgego_HostFuncWrapper)(void *, void *,
                                           WasmEdge_MemoryInstanceContext *,
                                           const WasmEdge_Value *,
                                           const uint32_t, WasmEdge_Value *,
                                           const uint32_t);

WasmEdge_Result
wasmedgego_HostFuncInvoke(void *Func, void *Data,
                          WasmEdge_MemoryInstanceContext *MemCxt,
                          const WasmEdge_Value *Params, const uint32_t ParamLen,
                          WasmEdge_Value *Returns, const uint32_t ReturnLen);

typedef uint32_t (*wasmedgego_GetExport)(const WasmEdge_ModuleInstanceContext *,
                                         WasmEdge_String *, const uint32_t);
typedef uint32_t (*wasmedgego_GetRegExport)(
    const WasmEdge_ModuleInstanceContext *, WasmEdge_String, WasmEdge_String *,
    const uint32_t);

uint32_t wasmedgego_WrapListExport(wasmedgego_GetExport F,
                                   const WasmEdge_ModuleInstanceContext *Cxt,
                                   WasmEdge_String *Names, const uint32_t Len) {
  return F(Cxt, Names, Len);
}
*/
import "C"
import (
	"errors"
	"reflect"
	"unsafe"
)

type Module struct {
	_inner     *C.WasmEdge_ModuleInstanceContext
	_hostfuncs []uint
	_own       bool
}

type Function struct {
	_inner *C.WasmEdge_FunctionInstanceContext
	_index uint
	_own   bool
}

type Table struct {
	_inner *C.WasmEdge_TableInstanceContext
	_own   bool
}

type Memory struct {
	_inner *C.WasmEdge_MemoryInstanceContext
	_own   bool
}

type Global struct {
	_inner *C.WasmEdge_GlobalInstanceContext
	_own   bool
}

func NewModule(modname string) *Module {
	module := C.WasmEdge_ModuleInstanceCreate(toWasmEdgeStringWrap(modname))
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasiModule(args []string, envs []string, preopens []string) *Module {
	cargs := toCStringArray(args)
	cenvs := toCStringArray(envs)
	cpreopens := toCStringArray(preopens)
	var ptrargs *(*C.char) = nil
	var ptrenvs *(*C.char) = nil
	var ptrpreopens *(*C.char) = nil
	if len(cargs) > 0 {
		ptrargs = &cargs[0]
	}
	if len(cenvs) > 0 {
		ptrenvs = &cenvs[0]
	}
	if len(cpreopens) > 0 {
		ptrpreopens = &cpreopens[0]
	}

	module := C.WasmEdge_ModuleInstanceCreateWASI(
		ptrargs, C.uint32_t(len(cargs)),
		ptrenvs, C.uint32_t(len(cenvs)),
		ptrpreopens, C.uint32_t(len(cpreopens)))

	freeCStringArray(cargs)
	freeCStringArray(cenvs)
	freeCStringArray(cpreopens)

	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func (self *Module) InitWasi(args []string, envs []string, preopens []string) {
	cargs := toCStringArray(args)
	cenvs := toCStringArray(envs)
	cpreopens := toCStringArray(preopens)
	var ptrargs *(*C.char) = nil
	var ptrenvs *(*C.char) = nil
	var ptrpreopens *(*C.char) = nil
	if len(cargs) > 0 {
		ptrargs = &cargs[0]
	}
	if len(cenvs) > 0 {
		ptrenvs = &cenvs[0]
	}
	if len(cpreopens) > 0 {
		ptrpreopens = &cpreopens[0]
	}

	C.WasmEdge_ModuleInstanceInitWASI(self._inner,
		ptrargs, C.uint32_t(len(cargs)),
		ptrenvs, C.uint32_t(len(cenvs)),
		ptrpreopens, C.uint32_t(len(cpreopens)))

	freeCStringArray(cargs)
	freeCStringArray(cenvs)
	freeCStringArray(cpreopens)
}

func (self *Module) WasiGetExitCode() uint {
	return uint(C.WasmEdge_ModuleInstanceWASIGetExitCode(self._inner))
}

func NewWasiNNModule() *Module {
	module := C.WasmEdge_ModuleInstanceCreateWasiNN()
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasiCryptoCommonModule() *Module {
	module := C.WasmEdge_ModuleInstanceCreateWasiCryptoCommon()
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasiCryptoAsymmetricCommonModule() *Module {
	module := C.WasmEdge_ModuleInstanceCreateWasiCryptoAsymmetricCommon()
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasiCryptoKxModule() *Module {
	module := C.WasmEdge_ModuleInstanceCreateWasiCryptoKx()
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasiCryptoSignaturesModule() *Module {
	module := C.WasmEdge_ModuleInstanceCreateWasiCryptoSignatures()
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasiCryptoSymmetricModule() *Module {
	module := C.WasmEdge_ModuleInstanceCreateWasiCryptoSymmetric()
	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func NewWasmEdgeProcessModule(allowedcmds []string, allowall bool) *Module {
	ccmds := toCStringArray(allowedcmds)
	var ptrcmds *(*C.char) = nil
	if len(ccmds) > 0 {
		ptrcmds = &ccmds[0]
	}

	module := C.WasmEdge_ModuleInstanceCreateWasmEdgeProcess(ptrcmds, C.uint32_t(len(ccmds)), C.bool(allowall))

	freeCStringArray(ccmds)

	if module == nil {
		return nil
	}
	return &Module{_inner: module, _own: true}
}

func (self *Module) InitWasmEdgeProcess(allowedcmds []string, allowall bool) {
	ccmds := toCStringArray(allowedcmds)
	var ptrcmds *(*C.char) = nil
	if len(ccmds) > 0 {
		ptrcmds = &ccmds[0]
	}

	C.WasmEdge_ModuleInstanceInitWasmEdgeProcess(ptrcmds, C.uint32_t(len(ccmds)), C.bool(allowall))

	freeCStringArray(ccmds)
}

func (self *Module) AddFunction(name string, inst *Function) {
	C.WasmEdge_ModuleInstanceAddFunction(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	self._hostfuncs = append(self._hostfuncs, inst._index)
	inst._inner = nil
	inst._own = false
}

func (self *Module) AddTable(name string, inst *Table) {
	C.WasmEdge_ModuleInstanceAddTable(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	inst._inner = nil
	inst._own = false
}

func (self *Module) AddMemory(name string, inst *Memory) {
	C.WasmEdge_ModuleInstanceAddMemory(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	inst._inner = nil
	inst._own = false
}

func (self *Module) AddGlobal(name string, inst *Global) {
	C.WasmEdge_ModuleInstanceAddGlobal(self._inner, toWasmEdgeStringWrap(name), inst._inner)
	inst._inner = nil
	inst._own = false
}

func (self *Module) getExports(exportlen C.uint32_t, getfunc C.wasmedgego_GetExport) []string {
	cnames := make([]C.WasmEdge_String, int(exportlen))
	if int(exportlen) > 0 {
		C.wasmedgego_WrapListExport(getfunc, self._inner, &cnames[0], exportlen)
	}
	names := make([]string, int(exportlen))
	for i := 0; i < int(exportlen); i++ {
		names[i] = fromWasmEdgeString(cnames[i])
	}
	return names
}

func (self *Module) FindFunction(name string) *Function {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_ModuleInstanceFindFunction(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Function{_inner: cinst, _own: false}
}

func (self *Module) FindTable(name string) *Table {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_ModuleInstanceFindTable(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Table{_inner: cinst, _own: false}
}

func (self *Module) FindMemory(name string) *Memory {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_ModuleInstanceFindMemory(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Memory{_inner: cinst, _own: false}
}

func (self *Module) FindGlobal(name string) *Global {
	cname := toWasmEdgeStringWrap(name)
	cinst := C.WasmEdge_ModuleInstanceFindGlobal(self._inner, cname)
	if cinst == nil {
		return nil
	}
	return &Global{_inner: cinst, _own: false}
}

func (self *Module) ListFunction() []string {
	return self.getExports(
		C.WasmEdge_ModuleInstanceListFunctionLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_ModuleInstanceListFunction),
	)
}

func (self *Module) ListTable() []string {
	return self.getExports(
		C.WasmEdge_ModuleInstanceListTableLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_ModuleInstanceListTable),
	)
}

func (self *Module) ListMemory() []string {
	return self.getExports(
		C.WasmEdge_ModuleInstanceListMemoryLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_ModuleInstanceListMemory),
	)
}

func (self *Module) ListGlobal() []string {
	return self.getExports(
		C.WasmEdge_ModuleInstanceListGlobalLength(self._inner),
		C.wasmedgego_GetExport(C.WasmEdge_ModuleInstanceListGlobal),
	)
}

func (self *Module) Release() {
	if self._own {
		for _, idx := range self._hostfuncs {
			hostfuncMgr.del(idx)
		}
		self._hostfuncs = []uint{}
		C.WasmEdge_ModuleInstanceDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func NewFunction(ftype *FunctionType, fn hostFunctionSignature, additional interface{}, cost uint) *Function {
	if ftype == nil {
		return nil
	}

	index := hostfuncMgr.add(fn, additional)
	function := C.WasmEdge_FunctionInstanceCreateBinding(
		ftype._inner,
		C.wasmedgego_HostFuncWrapper(C.wasmedgego_HostFuncInvoke),
		unsafe.Pointer(uintptr(index)),
		nil,
		C.uint64_t(cost))
	if function == nil {
		hostfuncMgr.del(index)
		return nil
	}
	return &Function{_inner: function, _index: index, _own: true}
}

func (self *Function) GetFunctionType() *FunctionType {
	return &FunctionType{
		_inner: C.WasmEdge_FunctionInstanceGetFunctionType(self._inner),
		_own:   false,
	}
}

func (self *Function) Release() {
	if self._own && self._inner != nil {
		C.WasmEdge_FunctionInstanceDelete(self._inner)
		hostfuncMgr.del(self._index)
	}
	self._inner = nil
	self._own = false
}

func NewTable(ttype *TableType) *Table {
	if ttype == nil {
		return nil
	}
	table := C.WasmEdge_TableInstanceCreate(ttype._inner)
	if table == nil {
		return nil
	}
	return &Table{_inner: table, _own: true}
}

func (self *Table) GetTableType() *TableType {
	return &TableType{
		_inner: C.WasmEdge_TableInstanceGetTableType(self._inner),
		_own:   false,
	}
}

func (self *Table) GetData(off uint) (interface{}, error) {
	cval := C.WasmEdge_Value{}
	res := C.WasmEdge_TableInstanceGetData(self._inner, &cval, C.uint32_t(off))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValue(cval), nil
}

func (self *Table) SetData(data interface{}, off uint) error {
	cval := toWasmEdgeValue(data)
	res := C.WasmEdge_TableInstanceSetData(self._inner, cval, C.uint32_t(off))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Table) GetSize() uint {
	return uint(C.WasmEdge_TableInstanceGetSize(self._inner))
}

func (self *Table) Grow(size uint) error {
	res := C.WasmEdge_TableInstanceGrow(self._inner, C.uint32_t(size))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Table) Release() {
	if self._own {
		C.WasmEdge_TableInstanceDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func NewMemory(mtype *MemoryType) *Memory {
	if mtype == nil {
		return nil
	}
	memory := C.WasmEdge_MemoryInstanceCreate(mtype._inner)
	if memory == nil {
		return nil
	}
	return &Memory{_inner: memory, _own: true}
}

func (self *Memory) GetMemoryType() *MemoryType {
	return &MemoryType{
		_inner: C.WasmEdge_MemoryInstanceGetMemoryType(self._inner),
		_own:   false,
	}
}

func (self *Memory) GetData(off uint, length uint) ([]byte, error) {
	p := C.WasmEdge_MemoryInstanceGetPointer(self._inner, C.uint32_t(off), C.uint32_t(length))
	if p == nil {
		return nil, errors.New("Failed get data pointer")
	}
	// Use SliceHeader to wrap the slice from cgo
	var r []byte
	s := (*reflect.SliceHeader)(unsafe.Pointer(&r))
	s.Cap = int(length)
	s.Len = int(length)
	s.Data = uintptr(unsafe.Pointer(p))
	return r, nil
}

func (self *Memory) SetData(data []byte, off uint, length uint) error {
	var ptrdata *C.uint8_t = nil
	if len(data) > 0 {
		ptrdata = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	res := C.WasmEdge_MemoryInstanceSetData(self._inner, ptrdata, C.uint32_t(off), C.uint32_t(length))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Memory) GetPageSize() uint {
	return uint(C.WasmEdge_MemoryInstanceGetPageSize(self._inner))
}

func (self *Memory) GrowPage(size uint) error {
	res := C.WasmEdge_MemoryInstanceGrowPage(self._inner, C.uint32_t(size))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *Memory) Release() {
	if self._own {
		C.WasmEdge_MemoryInstanceDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}

func NewGlobal(gtype *GlobalType, val interface{}) *Global {
	if gtype == nil {
		return nil
	}
	cval := toWasmEdgeValue(val)
	global := C.WasmEdge_GlobalInstanceCreate(gtype._inner, cval)
	if global == nil {
		return nil
	}
	return &Global{_inner: global, _own: true}
}

func (self *Global) GetGlobalType() *GlobalType {
	return &GlobalType{
		_inner: C.WasmEdge_GlobalInstanceGetGlobalType(self._inner),
		_own:   false,
	}
}

func (self *Global) GetValue() interface{} {
	cval := C.WasmEdge_GlobalInstanceGetValue(self._inner)
	return fromWasmEdgeValue(cval)
}

func (self *Global) SetValue(val interface{}) {
	C.WasmEdge_GlobalInstanceSetValue(self._inner, toWasmEdgeValue(val))
}

func (self *Global) Release() {
	if self._own {
		C.WasmEdge_GlobalInstanceDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
