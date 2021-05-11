package wasmedge

/*
#include <wasmedge.h>
typedef void (*wasmedgego_HostFuncWrapper)
  (void *, void *, WasmEdge_MemoryInstanceContext *, const WasmEdge_Value *, const uint32_t, WasmEdge_Value *, const uint32_t);

WasmEdge_Result wasmedgego_HostFuncInvoke(void *Func, void *Data,
                                  WasmEdge_MemoryInstanceContext *MemCxt,
                                  const WasmEdge_Value *Params, const uint32_t ParamLen,
                                  WasmEdge_Value *Returns, const uint32_t ReturnLen);
*/
import "C"
import (
	"reflect"
	"sync"
	"unsafe"
)

type HostFunction struct {
	_inner *C.WasmEdge_HostFunctionContext
	_index uint
}

type hostFunctionSignature func(data interface{}, mem *Memory, params []interface{}) ([]interface{}, Result)

type hostFunctionManager struct {
	mu sync.Mutex
	// Valid next index of map. Use and increase this index when gc is empty.
	idx uint
	// Recycled entries of map. Use entry in this slide when allocate a new host function.
	gc    []uint
	data  map[uint]interface{}
	funcs map[uint]hostFunctionSignature
}

func (self *hostFunctionManager) add(hostfunc hostFunctionSignature) uint {
	self.mu.Lock()
	defer self.mu.Unlock()

	var realidx uint
	if len(self.gc) > 0 {
		realidx = self.gc[len(self.gc)-1]
		self.gc = self.gc[0:]
	} else {
		realidx = self.idx
		self.idx++
	}
	self.funcs[realidx] = hostfunc
	self.data[realidx] = nil
	return realidx
}

func (self *hostFunctionManager) get(i uint) (hostFunctionSignature, interface{}) {
	self.mu.Lock()
	defer self.mu.Unlock()
	return self.funcs[i], self.data[i]
}

func (self *hostFunctionManager) del(i uint) {
	self.mu.Lock()
	defer self.mu.Unlock()
	delete(self.funcs, i)
	delete(self.data, i)
	self.gc = append(self.gc, i)
}

var hostfuncMgr = hostFunctionManager{
	idx:   0,
	data:  make(map[uint]interface{}),
	funcs: make(map[uint]hostFunctionSignature),
}

//export wasmedgego_HostFuncInvokeImpl
func wasmedgego_HostFuncInvokeImpl(fn uintptr, data *C.void, mem *C.WasmEdge_MemoryInstanceContext, params *C.WasmEdge_Value, paramlen C.uint32_t, returns *C.WasmEdge_Value, returnlen C.uint32_t) C.WasmEdge_Result {
	gomem := &Memory{
		_inner: mem,
	}

	goparams := make([]interface{}, uint(paramlen))
	var cparams []C.WasmEdge_Value
	if paramlen > 0 {
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&cparams)))
		sliceHeader.Cap = int(paramlen)
		sliceHeader.Len = int(paramlen)
		sliceHeader.Data = uintptr(unsafe.Pointer(params))
		for i := 0; i < int(paramlen); i++ {
			goparams[i] = fromWasmEdgeValue(cparams[i], cparams[i].Type)
			if cparams[i].Type == C.WasmEdge_ValType_ExternRef && !goparams[i].(ExternRef)._valid {
				panic("External reference is released")
			}
		}
	}

	gofunc, godata := hostfuncMgr.get(uint(fn))
	goreturns, err := gofunc(godata, gomem, goparams)

	var creturns []C.WasmEdge_Value
	if returnlen > 0 && goreturns != nil {
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&creturns)))
		sliceHeader.Cap = int(returnlen)
		sliceHeader.Len = int(returnlen)
		sliceHeader.Data = uintptr(unsafe.Pointer(returns))
		for i, val := range goreturns {
			if i < int(returnlen) {
				creturns[i] = toWasmEdgeValue(val)
			}
		}
	}

	return C.WasmEdge_Result{Code: C.uint8_t(err.code)}
}

func NewHostFunction(functype *FunctionType, fn hostFunctionSignature, cost uint) *HostFunction {
	cftype := toWasmEdgeFunctionType(functype)
	defer C.WasmEdge_FunctionTypeDelete(cftype)
	self := &HostFunction{
		_inner: nil,
		_index: 0,
	}

	self._index = hostfuncMgr.add(fn)
	chostfunc := C.WasmEdge_HostFunctionCreateBinding(cftype, C.wasmedgego_HostFuncWrapper(C.wasmedgego_HostFuncInvoke), unsafe.Pointer(uintptr(self._index)), C.uint64_t(cost))
	if chostfunc == nil {
		hostfuncMgr.del(self._index)
		return nil
	}
	self._inner = chostfunc
	return self
}

func (self *HostFunction) Delete() {
	if self._inner != nil {
		C.WasmEdge_HostFunctionDelete(self._inner)
		self._inner = nil
		hostfuncMgr.del(self._index)
	}
}
