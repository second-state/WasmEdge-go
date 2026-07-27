package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"unsafe"
)

type executionContextBinding struct {
	ctx context.Context
}

var executionContexts sync.Map // map[unsafe.Pointer]*executionContextBinding

func bindExecutionContext(
	executor unsafe.Pointer,
	ctx context.Context,
) (func(), error) {
	if ctx == nil {
		return func() {}, nil
	}
	entry := &executionContextBinding{ctx: ctx}
	if _, loaded := executionContexts.LoadOrStore(executor, entry); loaded {
		return nil, fmt.Errorf("executor already has a bound context: %w", ErrInUse)
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			executionContexts.CompareAndDelete(executor, entry)
		})
	}, nil
}

func contextForCallFrame(frame *C.WasmEdge_CallingFrameContext) context.Context {
	if frame == nil {
		return context.Background()
	}
	executor := unsafe.Pointer(C.WasmEdge_CallingFrameGetExecutor(frame))
	if raw, ok := executionContexts.Load(executor); ok {
		if entry, ok := raw.(*executionContextBinding); ok {
			return entry.ctx
		}
	}
	return context.Background()
}

// CallContext is the calling frame passed to a host function. It is valid
// only for the duration of that host call; escaping it (or anything
// borrowed from it) is a programmer error and panics on use.
type CallContext struct {
	ptr *C.WasmEdge_CallingFrameContext
	ctx context.Context

	// scope is shared by every value copy of CallContext. Keeping the atomic
	// inline would copy its last observed value, allowing a copy made during
	// the callback to remain apparently valid after the original expired.
	scope *callContextScope
}

type callContextScope struct {
	valid atomic.Bool
}

func newCallContext(
	ptr *C.WasmEdge_CallingFrameContext,
	ctx context.Context,
) *CallContext {
	scope := &callContextScope{}
	scope.valid.Store(true)
	return &CallContext{ptr: ptr, ctx: ctx, scope: scope}
}

func (c *CallContext) expire() {
	if c != nil && c.scope != nil {
		c.scope.valid.Store(false)
	}
}

func (c *CallContext) mustValid() {
	if c == nil || c.scope == nil || !c.scope.valid.Load() {
		panic("wasmedge: CallContext escaped its host function call")
	}
}

func (c *CallContext) assertAlive() { c.mustValid() }

// Context returns the context associated with an InvokeContext, ExecuteContext,
// or RunContext call. It is context.Background for synchronous and manually
// managed asynchronous execution. Host functions doing blocking Go work
// should select on Context().Done() so cancellation can settle promptly.
func (c *CallContext) Context() context.Context {
	c.mustValid()
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

// Executor returns the executor driving this call (borrowed).
func (c *CallContext) Executor() *Executor {
	c.mustValid()
	return borrowedExecutor(C.WasmEdge_CallingFrameGetExecutor(c.ptr), c)
}

// Module returns the module instance of the calling frame (borrowed). It
// may be nil when the host function is invoked through Executor.Invoke
// directly rather than from WASM code.
func (c *CallContext) Module() *Module {
	c.mustValid()
	return borrowedModule(C.WasmEdge_CallingFrameGetModuleInstance(c.ptr), c)
}

// Memory returns the memory instance at idx of the calling module
// (borrowed), or nil if there is none. Index 0 is the module's default
// memory.
func (c *CallContext) Memory(idx uint32) *Memory {
	c.mustValid()
	return borrowedMemory(C.WasmEdge_CallingFrameGetMemoryInstance(c.ptr, C.uint32_t(idx)), c)
}
