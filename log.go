package wasmedge

// #include "shims.h"
import "C"

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"
)

// LogLevel is a WasmEdge engine log severity.
type LogLevel int32

const (
	LogLevelTrace    LogLevel = C.WasmEdge_LogLevel_Trace
	LogLevelDebug    LogLevel = C.WasmEdge_LogLevel_Debug
	LogLevelInfo     LogLevel = C.WasmEdge_LogLevel_Info
	LogLevelWarn     LogLevel = C.WasmEdge_LogLevel_Warn
	LogLevelError    LogLevel = C.WasmEdge_LogLevel_Error
	LogLevelCritical LogLevel = C.WasmEdge_LogLevel_Critical
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelTrace:
		return "trace"
	case LogLevelDebug:
		return "debug"
	case LogLevelInfo:
		return "info"
	case LogLevelWarn:
		return "warn"
	case LogLevelError:
		return "error"
	case LogLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// LogMessage is one engine log record delivered to a callback installed with
// SetLogCallback.
type LogMessage struct {
	Message    string
	LoggerName string
	Level      LogLevel
	Time       time.Time
	ThreadID   uint64
}

// SetLogLevel sets the process-wide engine log filter.
func SetLogLevel(l LogLevel) {
	C.WasmEdge_LogSetLevel(C.WasmEdge_LogLevel(l))
}

// SetLogOff disables engine logging entirely.
func SetLogOff() {
	C.WasmEdge_LogOff()
}

// logCallback holds the installed Go callback. Process-wide, like the
// underlying C API.
var logCallback atomic.Pointer[func(LogMessage)]

// SetLogCallback routes engine log records to cb instead of the engine's
// default logger. Passing nil restores the default logger. The callback may
// be invoked concurrently from arbitrary threads and must not panic (panics
// are contained and dropped).
func SetLogCallback(cb func(LogMessage)) {
	if cb == nil {
		logCallback.Store(nil)
		C.wasmedgego_setLogCallback(0)
		return
	}
	logCallback.Store(&cb)
	C.wasmedgego_setLogCallback(1)
}

// RouteLogsToSlog forwards engine logs to a slog.Logger.
//
// TODO(intern-medium): A11 — finish the slog bridge: map LogLevelTrace and
// LogLevelCritical onto slog levels (slog has no direct equivalents; use
// custom levels below Debug / above Error), attach LoggerName and ThreadID
// as attrs (already sketched below), preserve msg.Time via slog.Record, and
// add log_test.go asserting a record round-trip through a slog.Handler test
// double. Pattern: the switch in LogLevel.String above; test pattern:
// errors_test.go table tests.
func RouteLogsToSlog(l *slog.Logger) {
	if l == nil {
		SetLogCallback(nil)
		return
	}
	SetLogCallback(func(m LogMessage) {
		lvl := slog.LevelInfo
		switch m.Level {
		case LogLevelDebug:
			lvl = slog.LevelDebug
		case LogLevelWarn:
			lvl = slog.LevelWarn
		case LogLevelError, LogLevelCritical:
			lvl = slog.LevelError
		}
		l.LogAttrs(context.Background(), lvl, m.Message,
			slog.String("logger", m.LoggerName),
			slog.Uint64("thread_id", m.ThreadID))
	})
}
