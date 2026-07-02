package wasmedge

// This file holds every //export used as a C callback target. Per cgo rules
// its preamble may only contain declarations; the C-side adapters live in
// shims.c.

// #include <wasmedge/wasmedge.h>
import "C"

import "time"

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
