package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"context"
	"runtime"
)

// Execution is an in-flight asynchronous WASM invocation (the C API's
// WasmEdge_Async). Exactly one of Wait or Close must be called eventually;
// Wait consumes the execution and releases the underlying object.
type Execution struct {
	ptr  *C.WasmEdge_Async
	life lifetime
}

func newExecution(ptr *C.WasmEdge_Async) *Execution {
	e := &Execution{ptr: ptr}
	arm(e, &e.life, "Execution", func() { C.WasmEdge_AsyncDelete(ptr) })
	return e
}

// Wait blocks until the invocation finishes, returns its results, and
// releases the execution. Calling Wait after Cancel reports the
// interruption as an error (ErrCodeInterrupted).
func (e *Execution) Wait() ([]Value, error) {
	defer runtime.KeepAlive(e)
	C.WasmEdge_AsyncWait(e.ptr)
	n := C.WasmEdge_AsyncGetReturnsLength(e.ptr)
	buf := make([]C.WasmEdge_Value, max(int(n), 1))
	err := newResult(C.WasmEdge_AsyncGet(e.ptr, &buf[0], n))
	_ = e.Close()
	if err != nil {
		return nil, err
	}
	return unpackValues(buf[:n]), nil
}

// Cancel interrupts the invocation. It is safe to call at any point,
// including after completion; a canceled Wait reports ErrCodeInterrupted.
func (e *Execution) Cancel() {
	defer runtime.KeepAlive(e)
	C.WasmEdge_AsyncCancel(e.ptr)
}

// TODO(intern-medium): A10 — bind the remaining Async APIs on this type:
//
//	WaitFor(d time.Duration) bool -> WasmEdge_AsyncWaitFor (true = finished)
//
// WaitFor must NOT release the object (Wait/Close still required), which is
// exactly the behavior TestExecuteAsync should pin down: WaitFor(0) on the
// Loop fixture returns false, then Cancel + Wait returns the interruption.
// Pattern: Wait above; extend TestVMExecuteAsync in vm_test.go.

// Close abandons the execution without collecting results. Prefer Wait;
// use Close only on error paths. No-op after Wait or a prior Close.
func (e *Execution) Close() error {
	ptr := e.ptr
	return e.life.close(func() { C.WasmEdge_AsyncDelete(ptr) })
}

// waitContext couples an Execution to ctx: when ctx fires first the
// invocation is canceled and ctx's error is returned; otherwise the
// results pass through.
func waitContext(ctx context.Context, e *Execution) ([]Value, error) {
	if ctx.Done() == nil {
		return e.Wait()
	}
	if err := ctx.Err(); err != nil {
		e.Cancel()
		_, _ = e.Wait()
		return nil, err
	}

	watcherDone := make(chan struct{})
	waitDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			e.Cancel()
		case <-waitDone:
		}
	}()

	vals, err := e.Wait()
	close(waitDone)
	<-watcherDone

	if ctxErr := ctx.Err(); ctxErr != nil && err != nil {
		// The engine reports the interruption; the caller cares about
		// the context's reason (deadline vs cancellation).
		return nil, ctxErr
	}
	return vals, err
}

// InvokeContext calls a function instance like Invoke, but the call can be
// canceled through ctx; on cancellation the WASM execution is interrupted
// and ctx's error is returned.
func (e *Executor) InvokeContext(ctx context.Context, fn *Function, params ...Value) ([]Value, error) {
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(fn)
	cparams := packValues(params)
	ptr := C.WasmEdge_ExecutorAsyncInvoke(e.ptr, fn.ptr,
		valuesPtr(cparams), C.uint32_t(len(cparams)))
	if ptr == nil {
		return nil, &Error{Category: ErrCategoryWASM, Code: ErrCodeRuntimeError,
			Message: "async invocation failed to start"}
	}
	return waitContext(ctx, newExecution(ptr))
}
