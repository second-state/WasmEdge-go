package wasmedge

/*
#include <wasmedge/wasmedge.h>

typedef void (*wasmedgego_HostFuncWrapper)
  (void *, void *, WasmEdge_MemoryInstanceContext *, const WasmEdge_Value *, const uint32_t, WasmEdge_Value *, const uint32_t);

WasmEdge_Result wasmedgego_HostFuncInvoke(void *Func, void *Data,
                                  WasmEdge_MemoryInstanceContext *MemCxt,
                                  const WasmEdge_Value *Params, const uint32_t ParamLen,
                                  WasmEdge_Value *Returns, const uint32_t ReturnLen);
*/
import "C"
import "unsafe"

type Function struct {
	_inner  *C.WasmEdge_FunctionInstanceContext
	_ishost bool
	_index  uint
}

type Table struct {
	_inner *C.WasmEdge_TableInstanceContext
}

type Memory struct {
	_inner *C.WasmEdge_MemoryInstanceContext
}

type Global struct {
	_inner *C.WasmEdge_GlobalInstanceContext
}

func NewHostFunction(ftype *FunctionType, fn hostFunctionSignature, additional interface{}, cost uint) *Function {
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
	return &Function{
		_inner:  function,
		_ishost: true,
		_index:  index,
	}
}

func (self *Function) GetFunctionType() *FunctionType {
	ftype := &FunctionType{
		_inner: C.WasmEdge_FunctionInstanceGetFunctionType(self._inner),
	}
	return ftype
}

func (self *Function) Delete() {
	if self._ishost && self._inner != nil {
		C.WasmEdge_FunctionInstanceDelete(self._inner)
		self._inner = nil
		hostfuncMgr.del(self._index)
	}
}

func NewTable(ttype *TableType) *Table {
	if ttype == nil {
		return nil
	}
	table := C.WasmEdge_TableInstanceCreate(ttype._inner)
	if table == nil {
		return nil
	}
	return &Table{_inner: table}
}

func (self *Table) GetTableType() *TableType {
	return &TableType{
		_inner: C.WasmEdge_TableInstanceGetTableType(self._inner),
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

func (self *Table) Delete() {
	C.WasmEdge_TableInstanceDelete(self._inner)
	self._inner = nil
}

func NewMemory(mtype *MemoryType) *Memory {
	if mtype == nil {
		return nil
	}
	memory := C.WasmEdge_MemoryInstanceCreate(mtype._inner)
	if memory == nil {
		return nil
	}
	return &Memory{_inner: memory}
}

func (self *Memory) GetMemoryType() *MemoryType {
	return &MemoryType{
		_inner: C.WasmEdge_MemoryInstanceGetMemoryType(self._inner),
	}
}

func (self *Memory) GetData(off uint, length uint) ([]byte, error) {
	data := make([]byte, length)
	var ptrdata *C.uint8_t = nil
	if len(data) > 0 {
		ptrdata = (*C.uint8_t)(unsafe.Pointer(&data[0]))
	}
	res := C.WasmEdge_MemoryInstanceGetData(self._inner, ptrdata, C.uint32_t(off), C.uint32_t(length))
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}

	return data, nil
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

func (self *Memory) Delete() {
	C.WasmEdge_MemoryInstanceDelete(self._inner)
	self._inner = nil
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
	return &Global{_inner: global}
}

func (self *Global) GetGlobalType() *GlobalType {
	return &GlobalType{
		_inner: C.WasmEdge_GlobalInstanceGetGlobalType(self._inner),
	}
}

func (self *Global) GetValue() interface{} {
	cval := C.WasmEdge_GlobalInstanceGetValue(self._inner)
	return fromWasmEdgeValue(cval)
}

func (self *Global) SetValue(val interface{}) {
	C.WasmEdge_GlobalInstanceSetValue(self._inner, toWasmEdgeValue(val))
}

func (self *Global) Delete() {
	C.WasmEdge_GlobalInstanceDelete(self._inner)
	self._inner = nil
}
