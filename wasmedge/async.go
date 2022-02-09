package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"
import (
	"unsafe"
)

type Async struct {
	_inner *C.WasmEdge_Async
	_own   bool
}

func (self *Async) WaitFor(millisec int) bool {
	return bool(C.WasmEdge_AsyncWaitFor(self._inner, C.uint64_t(millisec)))
}

func (self *Async) Cancel() {
	C.WasmEdge_AsyncCancel(self._inner)
}

func (self *Async) GetResult() ([]interface{}, error) {
	arity := C.WasmEdge_AsyncGetReturnsLength(self._inner)
	creturns := make([]C.WasmEdge_Value, arity)
	var ptrreturns *C.WasmEdge_Value = nil
	if len(creturns) > 0 {
		ptrreturns = (*C.WasmEdge_Value)(unsafe.Pointer(&creturns[0]))
	}
	res := C.WasmEdge_AsyncGet(self._inner, ptrreturns, arity)
	if !C.WasmEdge_ResultOK(res) {
		return nil, newError(res)
	}
	return fromWasmEdgeValueSlide(creturns), nil
}

func (self *Async) Release() {
	if self._own {
		C.WasmEdge_AsyncDelete(self._inner)
	}
	self._inner = nil
	self._own = false
}
