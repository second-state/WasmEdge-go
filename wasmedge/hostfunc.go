package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"reflect"
	"sync"
	"unsafe"
)

type hostFunctionSignature func(data interface{}, callframe *CallingFrame, params []interface{}) ([]interface{}, Result)

type hostFunctionManager struct {
	mu sync.Mutex
	// Valid next index of map. Use and increase this index when gc is empty.
	idx uint
	// Recycled entries of map. Use entry in this slide when allocate a new host function.
	gc    []uint
	data  map[uint]interface{}
	funcs map[uint]hostFunctionSignature
}

func (self *hostFunctionManager) add(hostfunc hostFunctionSignature, hostdata interface{}) uint {
	self.mu.Lock()
	defer self.mu.Unlock()

	var realidx uint
	if len(self.gc) > 0 {
		realidx = self.gc[len(self.gc)-1]
		self.gc = self.gc[0 : len(self.gc)-1]
	} else {
		realidx = self.idx
		self.idx++
	}
	self.funcs[realidx] = hostfunc
	self.data[realidx] = hostdata
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
func wasmedgego_HostFuncInvokeImpl(fn uintptr, data *C.void, callframe *C.WasmEdge_CallingFrameContext, params *C.WasmEdge_Value, paramlen C.uint32_t, returns *C.WasmEdge_Value, returnlen C.uint32_t) C.WasmEdge_Result {
	gocallgrame := &CallingFrame{
		_inner: callframe,
	}

	goparams := make([]interface{}, uint(paramlen))
	var cparams []C.WasmEdge_Value
	if paramlen > 0 {
		sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&cparams)))
		sliceHeader.Cap = int(paramlen)
		sliceHeader.Len = int(paramlen)
		sliceHeader.Data = uintptr(unsafe.Pointer(params))
		for i := 0; i < int(paramlen); i++ {
			goparams[i] = fromWasmEdgeValue(cparams[i])
			if cparams[i].Type == C.WasmEdge_ValType_ExternRef && !goparams[i].(ExternRef)._valid {
				panic("External reference is released")
			}
		}
	}

	gofunc, godata := hostfuncMgr.get(uint(fn))
	goreturns, err := gofunc(godata, gocallgrame, goparams)

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

	return C.WasmEdge_Result{Code: C.uint32_t(err.code)}
}
