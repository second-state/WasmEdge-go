package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

// CallContext is the calling frame passed to a host function. It is valid
// only for the duration of that host call; escaping it (or anything
// borrowed from it) is a programmer error and panics on use.
type CallContext struct {
	ptr   *C.WasmEdge_CallingFrameContext
	valid bool
}

func (c *CallContext) mustValid() {
	if !c.valid {
		panic("wasmedge: CallContext escaped its host function call")
	}
}

// Executor returns the executor driving this call (borrowed).
func (c *CallContext) Executor() *Executor {
	c.mustValid()
	return borrowedExecutor(C.WasmEdge_CallingFrameGetExecutor(c.ptr))
}

// Module returns the module instance of the calling frame (borrowed). It
// may be nil when the host function is invoked through Executor.Invoke
// directly rather than from WASM code.
func (c *CallContext) Module() *Module {
	c.mustValid()
	return borrowedModule(C.WasmEdge_CallingFrameGetModuleInstance(c.ptr))
}

// Memory returns the memory instance at idx of the calling module
// (borrowed), or nil if there is none. Index 0 is the module's default
// memory.
func (c *CallContext) Memory(idx uint32) *Memory {
	c.mustValid()
	return borrowedMemory(C.WasmEdge_CallingFrameGetMemoryInstance(c.ptr, C.uint32_t(idx)))
}
