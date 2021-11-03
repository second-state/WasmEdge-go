package wasmedge

/*
#include <wasmedge/wasmedge.h>
size_t _GoStringLen(_GoString_ s);
const char *_GoStringPtr(_GoString_ s);
*/
import "C"
import (
	"runtime"
	"unsafe"
)

type VM struct {
	_inner *C.WasmEdge_VMContext
	_own   bool
}

type bindgen int

const (
	Bindgen_return_void  bindgen = iota
	Bindgen_return_i32   bindgen = iota
	Bindgen_return_i64   bindgen = iota
	Bindgen_return_array bindgen = iota
)

func NewVM() *VM {
	vm := C.WasmEdge_VMCreate(nil, nil)
	if vm == nil {
		return nil
	}
	res := &VM{_inner: vm, _own: true}
	runtime.SetFinalizer(res, (*VM).Release)
	return res
}

func NewVMWithConfig(conf *Configure) *VM {
	vm := C.WasmEdge_VMCreate(conf._inner, nil)
	if vm == nil {
		return nil
	}
	res := &VM{_inner: vm, _own: true}
	runtime.SetFinalizer(res, (*VM).Release)
	return res
}

func NewVMWithStore(store *Store) *VM {
	vm := C.WasmEdge_VMCreate(nil, store._inner)
	if vm == nil {
		return nil
	}
	res := &VM{_inner: vm, _own: true}
	runtime.SetFinalizer(res, (*VM).Release)
	return res
}

func NewVMWithConfigAndStore(conf *Configure, store *Store) *VM {
	vm := C.WasmEdge_VMCreate(conf._inner, store._inner)
	if vm == nil {
		return nil
	}
	res := &VM{_inner: vm, _own: true}
	runtime.SetFinalizer(res, (*VM).Release)
	return res
}

func (self *VM) RegisterWasmFile(modname string, path string) error {
	modstr := toWasmEdgeStringWrap(modname)
	var cpath = C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.WasmEdge_VMRegisterModuleFromFile(self._inner, modstr, cpath)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) RegisterWasmBuffer(modname string, buf []byte) error {
	modstr := toWasmEdgeStringWrap(modname)
	res := C.WasmEdge_VMRegisterModuleFromBuffer(self._inner, modstr, (*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) RegisterImport(imp *ImportObject) error {
	res := C.WasmEdge_VMRegisterModuleFromImport(self._inner, imp._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) RegisterAST(modname string, ast *AST) error {
	modstr := toWasmEdgeStringWrap(modname)
	res := C.WasmEdge_VMRegisterModuleFromASTModule(self._inner, modstr, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) runWasm(funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.WasmEdge_VMValidate(self._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	res = C.WasmEdge_VMInstantiate(self._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.Execute(funcname, params...)
}

func (self *VM) RunWasmFile(path string, funcname string, params ...interface{}) ([]interface{}, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.WasmEdge_VMLoadWasmFromFile(self._inner, cpath)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmBuffer(buf []byte, funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.WasmEdge_VMLoadWasmFromBuffer(self._inner, (*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmAST(ast *AST, funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.WasmEdge_VMLoadWasmFromASTModule(self._inner, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) LoadWasmFile(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.WasmEdge_VMLoadWasmFromFile(self._inner, cpath)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) LoadWasmBuffer(buf []byte) error {
	res := C.WasmEdge_VMLoadWasmFromBuffer(self._inner, (*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) LoadWasmAST(ast *AST) error {
	res := C.WasmEdge_VMLoadWasmFromASTModule(self._inner, ast._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) Validate() error {
	res := C.WasmEdge_VMValidate(self._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) Instantiate() error {
	res := C.WasmEdge_VMInstantiate(self._inner)
	if !C.WasmEdge_ResultOK(res) {
		return newError(res)
	}
	return nil
}

func (self *VM) Execute(funcname string, params ...interface{}) ([]interface{}, error) {
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, ftype.GetReturnsLength())
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_VMExecute(
		self._inner, funcstr,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns), nil
}

// Special execute function for running with wasm-bindgen.
func (self *VM) ExecuteBindgen(funcname string, rettype bindgen, params ...interface{}) (interface{}, error) {
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	cparams := toWasmEdgeValueSlideBindgen(self, rettype, nil, params...)
	creturns := make([]C.WasmEdge_Value, ftype.GetReturnsLength())
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_VMExecute(
		self._inner, funcstr,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlideBindgen(self, rettype, nil, creturns)
}

func (self *VM) ExecuteRegistered(modname string, funcname string, params ...interface{}) ([]interface{}, error) {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionTypeRegistered(modname, funcname)
	cparams := toWasmEdgeValueSlide(params...)
	creturns := make([]C.WasmEdge_Value, ftype.GetReturnsLength())
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_VMExecuteRegistered(
		self._inner, modstr, funcstr,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns), nil
}

// Special execute function for running with wasm-bindgen.
func (self *VM) ExecuteBindgenRegistered(modname string, funcname string, rettype bindgen, params ...interface{}) (interface{}, error) {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	cparams := toWasmEdgeValueSlideBindgen(self, rettype, &modname, params...)
	creturns := make([]C.WasmEdge_Value, ftype.GetReturnsLength())
	var ptrparams *C.WasmEdge_Value = nil
	var ptrreturns *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}

	res := C.WasmEdge_VMExecuteRegistered(
		self._inner, modstr, funcstr,
		ptrparams, C.uint32_t(len(cparams)),
		ptrreturns, C.uint32_t(len(creturns)))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlideBindgen(self, rettype, &modname, creturns)
}

func (self *VM) GetFunctionType(funcname string) *FunctionType {
	funcstr := toWasmEdgeStringWrap(funcname)
	cftype := C.WasmEdge_VMGetFunctionType(self._inner, funcstr)
	if cftype != nil {
		ftype := &FunctionType{
			_inner: cftype,
		}
		return ftype
	}
	return nil
}

func (self *VM) GetFunctionTypeRegistered(modname string, funcname string) *FunctionType {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	cftype := C.WasmEdge_VMGetFunctionTypeRegistered(self._inner, modstr, funcstr)
	if cftype != nil {
		ftype := &FunctionType{
			_inner: cftype,
		}
		return ftype
	}
	return nil
}

func (self *VM) Cleanup() {
	C.WasmEdge_VMCleanup(self._inner)
}

func (self *VM) GetFunctionList() ([]string, []*FunctionType) {
	funclen := C.WasmEdge_VMGetFunctionListLength(self._inner)
	cfnames := make([]C.WasmEdge_String, int(funclen))
	cftypes := make([]*C.WasmEdge_FunctionTypeContext, int(funclen))
	if int(funclen) > 0 {
		C.WasmEdge_VMGetFunctionList(self._inner, &cfnames[0], &cftypes[0], funclen)
	}
	fnames := make([]string, int(funclen))
	ftypes := make([]*FunctionType, int(funclen))
	for i := 0; i < int(funclen); i++ {
		fnames[i] = fromWasmEdgeString(cfnames[i])
		ftypes[i] = &FunctionType{_inner: cftypes[i]}
	}
	return fnames, ftypes
}

func (self *VM) GetImportObject(host HostRegistration) *ImportObject {
	ptr := C.WasmEdge_VMGetImportModuleContext(self._inner, C.enum_WasmEdge_HostRegistration(host))
	if ptr != nil {
		return &ImportObject{_inner: ptr, _own: false}
	}
	return nil
}

func (self *VM) GetStore() *Store {
	return &Store{_inner: C.WasmEdge_VMGetStoreContext(self._inner), _own: false}
}

func (self *VM) GetStatistics() *Statistics {
	return &Statistics{_inner: C.WasmEdge_VMGetStatisticsContext(self._inner), _own: false}
}

func (self *VM) Release() {
	if self._own {
		C.WasmEdge_VMDelete(self._inner)
	}
	runtime.SetFinalizer(self, nil)
	self._inner = nil
	self._own = false
}
