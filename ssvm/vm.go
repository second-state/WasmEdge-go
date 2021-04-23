package ssvm

/*
#include <ssvm.h>
size_t _GoStringLen(_GoString_ s);
const char *_GoStringPtr(_GoString_ s);
*/
import "C"
import (
	"io/ioutil"
	"os"
	"unsafe"
)

type VM struct {
	_inner *C.SSVM_VMContext
}

func NewVM() *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(nil, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithConfig(conf *Configure) *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(conf._inner, nil),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithStore(store *Store) *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(nil, store._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func NewVMWithConfigAndStore(conf *Configure, store *Store) *VM {
	self := &VM{
		_inner: C.SSVM_VMCreate(conf._inner, store._inner),
	}
	if self._inner == nil {
		return nil
	}
	return self
}

func (self *VM) RegisterWasmFile(modname string, path string) error {
	modstr := toSSVMStringWrap(modname)
	var cpath = C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return newError(C.SSVM_VMRegisterModuleFromFile(self._inner, modstr, cpath))
}

func (self *VM) RegisterWasmBuffer(modname string, buf []byte) error {
	modstr := toSSVMStringWrap(modname)
	return newError(C.SSVM_VMRegisterModuleFromBuffer(self._inner, modstr, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf))))
}

func (self *VM) RegisterImport(imp *ImportObject) error {
	return newError(C.SSVM_VMRegisterModuleFromImport(self._inner, imp._inner))
}

func (self *VM) RegisterAST(modname string, ast *AST) error {
	modstr := toSSVMStringWrap(modname)
	return newError(C.SSVM_VMRegisterModuleFromASTModule(self._inner, modstr, ast._inner))
}

func (self *VM) runWasm(funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.SSVM_VMValidate(self._inner)
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	res = C.SSVM_VMInstantiate(self._inner)
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return self.Execute(funcname, params...)
}

func (self *VM) RunWasmFile(path string, funcname string, params ...interface{}) ([]interface{}, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.SSVM_VMLoadWasmFromFile(self._inner, cpath)
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmFileWithDataAndWASI(path string, funcname string, data []byte,
	args []string, envp []string, dirs []string, preopens []string, params ...interface{}) ([]interface{}, error) {
	/// Create a temp file
	tmpf, _ := ioutil.TempFile("", "tmp.*.bin")
	defer os.Remove(tmpf.Name())
	tmpf.Write(data)
	tmpf.Close()

	/// Init WASI (test)
	var wasi = self.GetImportObject(WASI)
	if wasi != nil {
		wasi.InitWasi(
			args, /// The args
			append(envp, "SSVM_DATA_TO_CALLEE="+tmpf.Name()), /// The envs
			append(dirs, "/tmp:/tmp"),                        /// The mapping directories
			preopens,                                         /// The preopens will be empty
		)
	}

	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	res := C.SSVM_VMLoadWasmFromFile(self._inner, cpath)
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmBuffer(buf []byte, funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.SSVM_VMLoadWasmFromBuffer(self._inner, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf)))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) RunWasmAST(ast *AST, funcname string, params ...interface{}) ([]interface{}, error) {
	res := C.SSVM_VMLoadWasmFromASTModule(self._inner, ast._inner)
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return self.runWasm(funcname, params...)
}

func (self *VM) LoadWasmFile(path string) error {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return newError(C.SSVM_VMLoadWasmFromFile(self._inner, cpath))
}

func (self *VM) LoadWasmBuffer(buf []byte) error {
	return newError(C.SSVM_VMLoadWasmFromBuffer(self._inner, (*C.uint8_t)(unsafe.Pointer(&buf)), C.uint32_t(len(buf))))
}

func (self *VM) LoadWasmAST(ast *AST) error {
	return newError(C.SSVM_VMLoadWasmFromASTModule(self._inner, ast._inner))
}

func (self *VM) Validate() error {
	return newError(C.SSVM_VMValidate(self._inner))
}

func (self *VM) Instantiate() error {
	return newError(C.SSVM_VMInstantiate(self._inner))
}

func (self *VM) Execute(funcname string, params ...interface{}) ([]interface{}, error) {
	funcstr := toSSVMStringWrap(funcname)
	ftype := self.GetFunctionType(funcname)
	cparams := toSSVMValueSlide(params...)
	creturns := make([]C.SSVM_Value, len(ftype._returns))
	var ptrparams *C.SSVM_Value = nil
	var ptrreturns *C.SSVM_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.SSVM_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.SSVM_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.SSVM_VMExecute(self._inner, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(creturns)))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return fromSSVMValueSlide(creturns, ftype._returns), nil
}

func (self *VM) ExecuteRegistered(modname string, funcname string, params ...interface{}) ([]interface{}, error) {
	modstr := toSSVMStringWrap(modname)
	funcstr := toSSVMStringWrap(funcname)
	ftype := self.GetFunctionTypeRegistered(modname, funcname)
	cparams := toSSVMValueSlide(params...)
	creturns := make([]C.SSVM_Value, len(ftype._returns))
	var ptrparams *C.SSVM_Value = nil
	var ptrreturns *C.SSVM_Value = nil
	if len(cparams) > 0 {
		ptrparams = (*C.SSVM_Value)(unsafe.Pointer(&cparams[0]))
	}
	if len(creturns) > 0 {
		ptrreturns = (*C.SSVM_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.SSVM_VMExecuteRegistered(self._inner, modstr, funcstr, ptrparams, C.uint32_t(len(cparams)), ptrreturns, C.uint32_t(len(creturns)))
	if !C.SSVM_ResultOK(res) {
		return nil, newError(res)
	}
	return fromSSVMValueSlide(creturns, ftype._returns), nil
}

func (self *VM) GetFunctionType(funcname string) *FunctionType {
	funcstr := toSSVMStringWrap(funcname)
	cftype := C.SSVM_VMGetFunctionType(self._inner, funcstr)
	defer C.SSVM_FunctionTypeDelete(cftype)
	return fromSSVMFunctionType(cftype)
}

func (self *VM) GetFunctionTypeRegistered(modname string, funcname string) *FunctionType {
	modstr := toSSVMStringWrap(modname)
	funcstr := toSSVMStringWrap(funcname)
	cftype := C.SSVM_VMGetFunctionTypeRegistered(self._inner, modstr, funcstr)
	defer C.SSVM_FunctionTypeDelete(cftype)
	return fromSSVMFunctionType(cftype)
}

func (self *VM) Cleanup() {
	C.SSVM_VMCleanup(self._inner)
}

func (self *VM) GetFunctionList() ([]string, []*FunctionType) {
	funclen := C.SSVM_VMGetFunctionListLength(self._inner)
	cfnames := make([]C.SSVM_String, int(funclen))
	cftypes := make([]*C.SSVM_FunctionTypeContext, int(funclen))
	if int(funclen) > 0 {
		C.SSVM_VMGetFunctionList(self._inner, &cfnames[0], &cftypes[0], funclen)
	}
	fnames := make([]string, int(funclen))
	ftypes := make([]*FunctionType, int(funclen))
	for i := 0; i < int(funclen); i++ {
		fnames[i] = fromSSVMString(cfnames[i])
		C.SSVM_StringDelete(cfnames[i])
		ftypes[i] = fromSSVMFunctionType(cftypes[i])
		C.SSVM_FunctionTypeDelete(cftypes[i])
	}
	return fnames, ftypes
}

func (self *VM) GetImportObject(host HostRegistration) *ImportObject {
	ptr := C.SSVM_VMGetImportModuleContext(self._inner, C.enum_SSVM_HostRegistration(host))
	if ptr != nil {
		return &ImportObject{
			_inner: ptr,
		}
	}
	return nil
}

func (self *VM) GetStore() *Store {
	return &Store{
		_inner: C.SSVM_VMGetStoreContext(self._inner),
	}
}

func (self *VM) GetStatistics() *Statistics {
	return &Statistics{
		_inner: C.SSVM_VMGetStatisticsContext(self._inner),
	}
}

func (self *VM) Delete() {
	C.SSVM_VMDelete(self._inner)
	self._inner = nil
}
