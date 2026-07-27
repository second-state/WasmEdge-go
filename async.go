package wasmedge

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

// ErrCancellationRace reports the WasmEdge 0.17.1 shared stop-token race:
// cancellation was requested, but the native invocation completed without
// consuming the token. The originating VM/Executor is marked ErrUnusable.
var ErrCancellationRace = errors.New("wasmedge: cancellation raced native completion")

// Execution is an in-flight asynchronous WASM invocation (the C API's
// WasmEdge_Async). Exactly one of Wait or Close must be called eventually;
// Wait consumes the execution and releases the underlying object.
type Execution struct {
	ptr             *C.WasmEdge_Async
	life            lifetime
	resultOwner     any
	params          []Value
	release         func()
	poison          func()
	cancelRequested atomic.Bool
	terminal        atomic.Bool
	opMu            sync.Mutex
	opCond          *sync.Cond
	deleting        bool
	nativeOps       int
}

func newExecution(
	ptr *C.WasmEdge_Async,
	resultOwner any,
	params []Value,
	release func(),
	poison func(),
) *Execution {
	e := &Execution{
		ptr:         ptr,
		resultOwner: resultOwner,
		params:      append([]Value(nil), params...),
		release:     release,
		poison:      poison,
	}
	e.opCond = sync.NewCond(&e.opMu)
	arm(e, &e.life, "Execution", func() { C.WasmEdge_AsyncDelete(ptr) })
	return e
}

func (e *Execution) assertAlive() { e.life.assertAlive("Execution") }

func (e *Execution) beginTerminal() error {
	if !e.life.alive() {
		return ErrClosed
	}
	if !e.terminal.CompareAndSwap(false, true) {
		if !e.life.alive() {
			return ErrClosed
		}
		return ErrInUse
	}
	return nil
}

// beginNativeOp prevents a short observational/cancellation operation from
// racing WasmEdge_AsyncDelete. The terminal waiter deliberately does not hold
// this lock while blocked in WasmEdge_AsyncWait: Cancel must remain able to
// interrupt that wait.
func (e *Execution) beginNativeOp() error {
	e.opMu.Lock()
	defer e.opMu.Unlock()
	if e.deleting || !e.life.alive() {
		return ErrClosed
	}
	e.nativeOps++
	return nil
}

func (e *Execution) endNativeOp() {
	e.opMu.Lock()
	e.nativeOps--
	if e.nativeOps == 0 {
		e.opCond.Broadcast()
	}
	e.opMu.Unlock()
}

func (e *Execution) quiesceNativeOps() {
	e.opMu.Lock()
	e.deleting = true
	for e.nativeOps != 0 {
		e.opCond.Wait()
	}
	e.opMu.Unlock()
}

func (e *Execution) collect() ([]Value, error) {
	e.assertAlive()
	defer runtime.KeepAlive(e)
	C.WasmEdge_AsyncWait(e.ptr)
	n := C.WasmEdge_AsyncGetReturnsLength(e.ptr)
	buf := make([]C.WasmEdge_Value, max(int(n), 1))
	err := newResult(C.WasmEdge_AsyncGet(e.ptr, &buf[0], n))
	if err != nil {
		return nil, err
	}
	return unpackValuesOwned(buf[:n], e.resultOwner, e.params...), nil
}

// Wait blocks until the invocation finishes, returns its results, and
// releases the execution. Calling Wait after Cancel normally reports
// ErrCodeInterrupted; if cancellation raced native completion it reports
// ErrCancellationRace and poisons the originating VM/Executor.
func (e *Execution) Wait() ([]Value, error) {
	if err := e.beginTerminal(); err != nil {
		return nil, fmt.Errorf("wait execution: %w", err)
	}
	return e.waitClaimed()
}

func (e *Execution) waitClaimed() ([]Value, error) {
	vals, err := e.collect()
	releaseErr, err := e.finishDelete(err)
	if err != nil {
		return nil, err
	}
	return vals, releaseErr
}

// Cancel interrupts an open invocation. It may be called after native
// completion but not after Wait or Close has released the execution.
// It only requests interruption; Wait or Close must still consume the
// execution.
func (e *Execution) Cancel() error {
	if err := e.beginNativeOp(); err != nil {
		return fmt.Errorf("cancel execution: %w", err)
	}
	defer e.endNativeOp()
	defer runtime.KeepAlive(e)
	if bool(C.WasmEdge_AsyncWaitFor(e.ptr, 0)) {
		return nil
	}
	e.cancelRequested.Store(true)
	C.WasmEdge_AsyncCancel(e.ptr)
	return nil
}

// WaitFor waits for at most d and reports whether execution finished. It does
// not release the execution; call Wait to collect results or Close to abandon
// them. Durations are rounded up to the millisecond resolution of the C API.
func (e *Execution) WaitFor(d time.Duration) bool {
	e.assertAlive()
	if err := e.beginNativeOp(); err != nil {
		panic(fmt.Sprintf("wasmedge: Execution WaitFor: %v", err))
	}
	defer e.endNativeOp()
	defer runtime.KeepAlive(e)
	var milliseconds uint64
	if d > 0 {
		milliseconds = uint64(d / time.Millisecond)
		if d%time.Millisecond != 0 {
			milliseconds++
		}
	}
	return bool(C.WasmEdge_AsyncWaitFor(e.ptr, C.uint64_t(milliseconds)))
}

// Close abandons returned values, drains or cancels an unfinished invocation,
// and releases the execution. Prefer Wait; use Close on abandonment/error
// paths. No-op after Wait or a prior Close.
func (e *Execution) Close() error {
	if !e.life.alive() {
		return nil
	}
	if err := e.beginTerminal(); err != nil {
		if errors.Is(err, ErrClosed) {
			return nil
		}
		return fmt.Errorf("close execution: %w", err)
	}
	defer runtime.KeepAlive(e)

	var executionErr error
	if !e.WaitFor(0) {
		_ = e.Cancel()
	}
	// Freeze new observational/cancellation operations before deciding
	// whether a result must be collected. A concurrent Cancel may have begun
	// just before Close claimed the terminal path.
	e.quiesceNativeOps()
	// A prior Cancel may have raced native completion. Even when the async
	// object is complete now, collect its result once before deletion. Besides
	// detecting the shared native stop-token race, this consumes any private
	// Go host-error token. Tokens intentionally have no wall-clock expiry:
	// a caller may wait arbitrarily long before settling an Execution.
	_, collectedErr := e.collect()
	if e.cancelRequested.Load() {
		executionErr = collectedErr
	}
	releaseErr, executionErr := e.finishDelete(executionErr)
	if isInterrupted(executionErr) {
		executionErr = nil
	}
	if executionErr != nil {
		return executionErr
	}
	return releaseErr
}

func (e *Execution) finishDelete(executionErr error) (error, error) {
	e.quiesceNativeOps()
	// Classify before releasing the execution gate. A cancellation that raced
	// native completion leaves WasmEdge 0.17.1's shared stop token set, so the
	// originating VM/Executor must be poisoned before another call can start.
	executionErr = e.classifyCancellation(executionErr)
	ptr := e.ptr
	err := e.life.close(func() { C.WasmEdge_AsyncDelete(ptr) })
	if err == nil {
		if e.release != nil {
			e.release()
			e.release = nil
		}
		e.resultOwner = nil
		e.params = nil
		e.poison = nil
	}
	return err, executionErr
}

func isInterrupted(err error) bool {
	var native *Error
	return errors.As(err, &native) && native.Code == ErrCodeInterrupted
}

// classifyCancellation runs only after short Cancel/WaitFor operations have
// quiesced. That closes the narrow race in which AsyncGet observed completion
// immediately before a concurrent Cancel set the shared native stop token.
func (e *Execution) classifyCancellation(err error) error {
	if !e.cancelRequested.Load() || isInterrupted(err) ||
		errors.Is(err, ErrCancellationRace) {
		return err
	}
	if e.poison != nil {
		e.poison()
	}
	if err == nil {
		return ErrCancellationRace
	}
	return fmt.Errorf("%w: native result: %v", ErrCancellationRace, err)
}

// waitContext couples an Execution to ctx: when ctx fires first the
// invocation is canceled and ctx's error is returned; otherwise the
// results pass through.
func waitContext(ctx context.Context, e *Execution) ([]Value, error) {
	if err := e.beginTerminal(); err != nil {
		return nil, fmt.Errorf("wait context execution: %w", err)
	}
	if ctx.Done() == nil {
		return e.waitClaimed()
	}
	if err := ctx.Err(); err != nil {
		_ = e.Cancel()
		vals, executionErr := e.collect()
		releaseErr, executionErr := e.finishDelete(executionErr)
		if e.cancelRequested.Load() {
			return nil, err
		}
		if executionErr == nil {
			executionErr = releaseErr
		}
		return vals, executionErr
	}

	watcherDone := make(chan struct{})
	waitDone := make(chan struct{})
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			_ = e.Cancel()
		case <-waitDone:
		}
	}()

	vals, err := e.collect()
	close(waitDone)
	<-watcherDone
	releaseErr, err := e.finishDelete(err)

	if ctxErr := ctx.Err(); ctxErr != nil && e.cancelRequested.Load() {
		// The engine reports the interruption; the caller cares about
		// the context's reason (deadline vs cancellation).
		return nil, ctxErr
	}
	if err == nil {
		err = releaseErr
	}
	return vals, err
}

// InvokeAsync starts a manually managed asynchronous function invocation.
// The Executor permits one in-flight invocation at a time. Call exactly one
// of Execution.Wait or Execution.Close to release it.
func (e *Executor) InvokeAsync(fn *Function, params ...Value) (*Execution, error) {
	//nolint:staticcheck // A nil context is the internal no-cancellation sentinel.
	return e.invokeAsync(nil, fn, params...)
}

func (e *Executor) invokeAsync(
	ctx context.Context,
	fn *Function,
	params ...Value,
) (*Execution, error) {
	e.assertAlive()
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if fn == nil {
		return nil, fmt.Errorf("invoke requires a non-nil function: %w", ErrInvalidArgument)
	}
	fn.assertAlive()
	assertValueOwnersAlive(params)
	ft := fn.Type()
	if err := validateInvocationParams(ft, params); err != nil {
		return nil, err
	}
	cparams, paramCount, err := packValues(params)
	if err != nil {
		return nil, err
	}
	releaseGate, err := e.gate.begin()
	if err != nil {
		return nil, fmt.Errorf("invoke function: %w", err)
	}
	finishReferences, err := fn.prepareInvocationReferences(params)
	if err != nil {
		releaseGate()
		return nil, fmt.Errorf("retain invocation references: %w", err)
	}
	defer finishReferences(false)
	owners := append([]any{e, fn}, valueOwners(params)...)
	releaseLeases, err := acquireLeases(owners...)
	if err != nil {
		releaseGate()
		return nil, fmt.Errorf("lease async invocation: %w", err)
	}
	releaseContext, err := bindExecutionContext(unsafe.Pointer(e.ptr), ctx)
	if err != nil {
		releaseLeases()
		releaseGate()
		return nil, fmt.Errorf("bind invocation context: %w", err)
	}
	release := func() {
		releaseContext()
		releaseLeases()
		releaseGate()
	}
	defer runtime.KeepAlive(e)
	defer runtime.KeepAlive(fn)
	ptr := C.WasmEdge_ExecutorAsyncInvoke(e.ptr, fn.ptr,
		valuesPtr(cparams), paramCount)
	if ptr == nil {
		release()
		return nil, fmt.Errorf("start executor invocation: %w", ErrUnavailable)
	}
	finishReferences(true)
	return newExecution(ptr, fn, params, release, e.gate.poison), nil
}

// InvokeContext calls a function instance like Invoke, but the call can be
// canceled through ctx; on cancellation the WASM execution is interrupted
// and ctx's error is returned.
func (e *Executor) InvokeContext(ctx context.Context, fn *Function, params ...Value) ([]Value, error) {
	e.assertAlive()
	if ctx == nil {
		return nil, fmt.Errorf("invoke context is nil: %w", ErrInvalidArgument)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	execution, err := e.invokeAsync(ctx, fn, params...)
	if err != nil {
		return nil, err
	}
	return waitContext(ctx, execution)
}
