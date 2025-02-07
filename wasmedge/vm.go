package wasmedge

/*
#include <wasmedge/wasmedge.h>
#include <stdlib.h>

size_t _GoStringLen(_GoString_ s);
const char *_GoStringPtr(_GoString_ s);
*/
import "C"
import (
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
	return &VM{_inner: vm, _own: true}
}

func NewVMWithConfig(conf *Configure) *VM {
	vm := C.WasmEdge_VMCreate(conf._inner, nil)
	if vm == nil {
		return nil
	}
	return &VM{_inner: vm, _own: true}
}

func NewVMWithStore(store *Store) *VM {
	vm := C.WasmEdge_VMCreate(nil, store._inner)
	if vm == nil {
		return nil
	}
	return &VM{_inner: vm, _own: true}
}

func NewVMWithConfigAndStore(conf *Configure, store *Store) *VM {
	vm := C.WasmEdge_VMCreate(conf._inner, store._inner)
	if vm == nil {
		return nil
	}
	return &VM{_inner: vm, _own: true}
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
	cbytes := C.WasmEdge_BytesWrap((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	res := C.WasmEdge_VMRegisterModuleFromBytes(self._inner, modstr, cbytes)
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

func (self *VM) RegisterModule(module *Module) error {
	res := C.WasmEdge_VMRegisterModuleFromImport(self._inner, module._inner)
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
	cbytes := C.WasmEdge_BytesWrap((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	res := C.WasmEdge_VMLoadWasmFromBytes(self._inner, cbytes)
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

func (self *VM) AsyncRunWasmFile(path string, funcname string, params ...interface{}) *Async {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	funcstr := toWasmEdgeStringWrap(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	var ptrparams *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	async := C.WasmEdge_VMAsyncRunWasmFromFile(
		self._inner, cpath, funcstr, ptrparams, C.uint32_t(len(cparams)))
	if async == nil {
		return nil
	}
	return &Async{_inner: async, _own: true}
}

func (self *VM) AsyncRunWasmBuffer(buf []byte, funcname string, params ...interface{}) *Async {
	funcstr := toWasmEdgeStringWrap(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	var ptrparams *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	cbytes := C.WasmEdge_BytesWrap((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	async := C.WasmEdge_VMAsyncRunWasmFromBytes(self._inner, cbytes, funcstr, ptrparams, C.uint32_t(len(cparams)))
	if async == nil {
		return nil
	}
	return &Async{_inner: async, _own: true}
}

func (self *VM) AsyncRunWasmAST(ast *AST, funcname string, params ...interface{}) *Async {
	funcstr := toWasmEdgeStringWrap(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	var ptrparams *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	async := C.WasmEdge_VMAsyncRunWasmFromASTModule(
		self._inner, ast._inner, funcstr, ptrparams, C.uint32_t(len(cparams)))
	if async == nil {
		return nil
	}
	return &Async{_inner: async, _own: true}
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
	cbytes := C.WasmEdge_BytesWrap((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint32_t(len(buf)))
	res := C.WasmEdge_VMLoadWasmFromBytes(self._inner, cbytes)
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
	if ftype == nil {
		// If get function type failed, set as NULL and keep running to let the VM to handle the error.
		ftype = &FunctionType{_inner: nil, _own: false}
	}
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

func (self *VM) AsyncExecute(funcname string, params ...interface{}) *Async {
	funcstr := toWasmEdgeStringWrap(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	var ptrparams *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	async := C.WasmEdge_VMAsyncExecute(self._inner, funcstr, ptrparams, C.uint32_t(len(cparams)))
	if async == nil {
		return nil
	}
	return &Async{_inner: async, _own: true}
}

// Special execute function for running with wasm-bindgen.
func (self *VM) ExecuteBindgen(funcname string, rettype bindgen, params ...interface{}) (interface{}, error) {
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	if ftype == nil {
		// If get function type failed, set as NULL and keep running to let the VM to handle the error.
		ftype = &FunctionType{_inner: nil, _own: false}
	}
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
	if ftype == nil {
		// If get function type failed, set as NULL and keep running to let the VM to handle the error.
		ftype = &FunctionType{_inner: nil, _own: false}
	}
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

func (self *VM) AsyncExecuteRegistered(modname string, funcname string, params ...interface{}) *Async {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	cparams := toWasmEdgeValueSlide(params...)
	var ptrparams *C.WasmEdge_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.WasmEdge_Value)(unsafe.Pointer(&cparams[0]))
	}
	async := C.WasmEdge_VMAsyncExecuteRegistered(self._inner, modstr, funcstr, ptrparams, C.uint32_t(len(cparams)))
	if async == nil {
		return nil
	}
	return &Async{_inner: async, _own: true}
}

// Special execute function for running with wasm-bindgen.
func (self *VM) ExecuteBindgenRegistered(modname string, funcname string, rettype bindgen, params ...interface{}) (interface{}, error) {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	if ftype == nil {
		// If get function type failed, set as NULL and keep running to let the VM to handle the error.
		ftype = &FunctionType{_inner: nil, _own: false}
	}
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
	if cftype == nil {
		return nil
	}
	return &FunctionType{_inner: cftype, _own: false}
}

func (self *VM) GetFunctionTypeRegistered(modname string, funcname string) *FunctionType {
	modstr := toWasmEdgeStringWrap(modname)
	funcstr := toWasmEdgeStringWrap(funcname)
	cftype := C.WasmEdge_VMGetFunctionTypeRegistered(self._inner, modstr, funcstr)
	if cftype == nil {
		return nil
	}
	return &FunctionType{_inner: cftype, _own: false}
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

func (self *VM) GetImportModule(host HostRegistration) *Module {
	ptr := C.WasmEdge_VMGetImportModuleContext(self._inner, C.enum_WasmEdge_HostRegistration(host))
	if ptr != nil {
		return &Module{_inner: ptr, _own: false}
	}
	return nil
}

func (self *VM) GetActiveModule() *Module {
	ptr := C.WasmEdge_VMGetActiveModule(self._inner)
	if ptr != nil {
		return &Module{_inner: ptr, _own: false}
	}
	return nil
}

func (self *VM) GetRegisteredModule(name string) *Module {
	cname := toWasmEdgeStringWrap(name)
	ptr := C.WasmEdge_VMGetRegisteredModule(self._inner, cname)
	if ptr != nil {
		return &Module{_inner: ptr, _own: false}
	}
	return nil
}

func (self *VM) ListRegisteredModule() []string {
	modlen := C.WasmEdge_VMListRegisteredModuleLength(self._inner)
	cnames := make([]C.WasmEdge_String, int(modlen))
	if int(modlen) > 0 {
		C.WasmEdge_VMListRegisteredModule(self._inner, &cnames[0], modlen)
	}
	names := make([]string, int(modlen))
	for i := 0; i < int(modlen); i++ {
		names[i] = fromWasmEdgeString(cnames[i])
	}
	return names
}

func (self *VM) GetStore() *Store {
	return &Store{_inner: C.WasmEdge_VMGetStoreContext(self._inner), _own: false}
}

func (self *VM) GetLoader() *Loader {
	return &Loader{_inner: C.WasmEdge_VMGetLoaderContext(self._inner), _own: false}
}

func (self *VM) GetValidator() *Validator {
	return &Validator{_inner: C.WasmEdge_VMGetValidatorContext(self._inner), _own: false}
}

func (self *VM) GetExecutor() *Executor {
	return &Executor{_inner: C.WasmEdge_VMGetExecutorContext(self._inner), _own: false}
}

func (self *VM) GetStatistics() *Statistics {
	return &Statistics{_inner: C.WasmEdge_VMGetStatisticsContext(self._inner), _own: false}
}

func (self *VM) Release() {
	if self._own {
		C.WasmEdge_VMDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
