package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

type CallingFrame struct {
	_inner *C.WasmEdge_CallingFrameContext
}

func (self *CallingFrame) GetExecutor() *Executor {
	cinst := C.WasmEdge_CallingFrameGetExecutor(self._inner)
	if cinst == nil {
		return nil
	}
	return &Executor{_inner: cinst, _own: false}
}

func (self *CallingFrame) GetModule() *Module {
	cinst := C.WasmEdge_CallingFrameGetModuleInstance(self._inner)
	if cinst == nil {
		return nil
	}
	return &Module{_inner: cinst, _own: false}
}

func (self *CallingFrame) GetMemoryByIndex(idx int) *Memory {
	cinst := C.WasmEdge_CallingFrameGetMemoryInstance(self._inner, C.uint32_t(idx))
	if cinst == nil {
		return nil
	}
	return &Memory{_inner: cinst, _own: false}
}
