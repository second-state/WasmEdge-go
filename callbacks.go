package wasmedge

// This file holds every //export used as a C callback target. Per cgo rules
// its preamble may only contain declarations; the C-side adapters live in
// shims.c.

// #include <wasmedge/wasmedge.h>
import "C"

import (
	"fmt"
	"runtime/cgo"
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
// the cgo.Handle of the registered hostFuncEntry. The engine guarantees the
// params and returns arrays match the function type's arities.
//
//export wasmedgego_hostFuncInvoke
func wasmedgego_hostFuncInvoke(handle C.uintptr_t, frame *C.WasmEdge_CallingFrameContext,
	params *C.WasmEdge_Value, paramLen C.uint32_t,
	returns *C.WasmEdge_Value, returnLen C.uint32_t) C.WasmEdge_Result {

	err := func() (err error) {
		// A Go panic must never unwind into C: convert it into a
		// user-level failure and let WASM see a trap.
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("wasmedge: host function panicked: %v", r)
			}
		}()

		entry, ok := cgo.Handle(handle).Value().(*hostFuncEntry)
		if !ok {
			return fmt.Errorf("wasmedge: corrupt host function handle")
		}

		call := &CallContext{ptr: frame, valid: true}
		defer func() { call.valid = false }()

		in := unpackValues(unsafe.Slice(params, int(paramLen)))
		out, ferr := entry.fn(call, in)
		if ferr != nil {
			return ferr
		}
		if len(out) != int(returnLen) {
			return fmt.Errorf("wasmedge: host function returned %d values, type wants %d",
				len(out), int(returnLen))
		}
		if returnLen > 0 {
			dst := unsafe.Slice(returns, int(returnLen))
			for i, v := range out {
				dst[i] = v.raw
			}
		}
		return nil
	}()
	return toResult(err)
}

// wasmedgego_moduleDataFinalize releases the cgo.Handle pinning a module's
// host data; the engine invokes it (via shims.c) when the module instance
// is destroyed.
//
//export wasmedgego_moduleDataFinalize
func wasmedgego_moduleDataFinalize(handle C.uintptr_t) {
	defer func() { _ = recover() }()
	if handle != 0 {
		cgo.Handle(handle).Delete()
	}
}
