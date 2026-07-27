package wasmedge

// This file holds every //export used as a C callback target. Per cgo rules
// its preamble may only contain declarations; the C-side adapters live in
// shims.c.

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"
	"unsafe"
)

//export wasmedgego_logCallback
func wasmedgego_logCallback(msg *C.WasmEdge_LogMessage) {
	// The engine may log from arbitrary native threads and a Go panic must
	// never unwind into C, so contain everything.
	defer func() { _ = recover() }()

	cb := logCallback.Load()
	if cb == nil || msg == nil {
		return
	}
	(*cb)(LogMessage{
		Message:    goString(msg.Message),
		LoggerName: goString(msg.LoggerName),
		Level:      LogLevel(msg.Level),
		Time:       time.Unix(int64(msg.Time), 0),
		ThreadID:   uint64(msg.ThreadId),
	})
}

// wasmedgego_hostFuncInvoke is the single trampoline behind every host
// function: the engine calls the C shim (shims.c), which forwards here with
// a package-allocated opaque C token. The engine guarantees the
// params and returns arrays match the function type's arities.
//
//export wasmedgego_hostFuncInvoke
func wasmedgego_hostFuncInvoke(token unsafe.Pointer, frame *C.WasmEdge_CallingFrameContext,
	params *C.WasmEdge_Value, paramLen C.uint32_t,
	returns *C.WasmEdge_Value, returnLen C.uint32_t) (result C.WasmEdge_Result) {

	// This recovery is deliberately the outermost operation in the exported
	// callback. In particular it also covers toResult, whose errors.Is/As
	// traversal can invoke user-defined methods that panic.
	defer func() {
		if r := recover(); r != nil {
			result = hostPanicResult(r)
		}
	}()

	err := func() (err error) {
		// A Go panic must never unwind into C: convert it into a
		// user-level failure and let WASM see a trap.
		defer func() {
			if r := recover(); r != nil {
				err = &HostPanicError{Value: r, Stack: debug.Stack()}
			}
		}()

		entry, ok := hostFuncEntryForToken(token)
		if !ok {
			return fmt.Errorf("wasmedge: corrupt host function handle")
		}

		call := newCallContext(frame, contextForCallFrame(frame))
		defer call.expire()

		in := unpackValuesOwned(unsafe.Slice(params, int(paramLen)), call)
		out, ferr := entry.fn(call, in)
		if ferr != nil {
			return ferr
		}
		if len(out) != int(returnLen) {
			return fmt.Errorf("wasmedge: host function returned %d values, type wants %d",
				len(out), int(returnLen))
		}
		if err := entry.validateResults(out); err != nil {
			return err
		}
		if err := entry.retainResults(call, out); err != nil {
			return err
		}
		if returnLen > 0 {
			dst := unsafe.Slice(returns, int(returnLen))
			for i, v := range out {
				v.assertOwnerAlive()
				dst[i] = v.raw
			}
		}
		runtime.KeepAlive(out)
		return nil
	}()
	return toResult(err)
}

func hostPanicResult(value any) C.WasmEdge_Result {
	code := registerHostError(&HostPanicError{Value: value, Stack: debug.Stack()})
	return C.WasmEdge_ResultGen(
		C.WasmEdge_ErrCategory_UserLevelError,
		C.uint32_t(code),
	)
}

// wasmedgego_moduleDataFinalize resolves an opaque C token and releases the
// registry entry's internal cgo.Handle; only the C token crosses the callback
// boundary. The engine invokes it (via shims.c) when the module instance is
// destroyed.
//
//export wasmedgego_moduleDataFinalize
func wasmedgego_moduleDataFinalize(token unsafe.Pointer) {
	defer func() { _ = recover() }()
	finalizeModuleData(token)
}
