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

// Log level values include WasmEdge severities and their slog extensions.
const (
	LogLevelTrace    LogLevel = C.WasmEdge_LogLevel_Trace
	LogLevelDebug    LogLevel = C.WasmEdge_LogLevel_Debug
	LogLevelInfo     LogLevel = C.WasmEdge_LogLevel_Info
	LogLevelWarn     LogLevel = C.WasmEdge_LogLevel_Warn
	LogLevelError    LogLevel = C.WasmEdge_LogLevel_Error
	LogLevelCritical LogLevel = C.WasmEdge_LogLevel_Critical

	// SlogLevelTrace and SlogLevelCritical preserve WasmEdge's severities on
	// slog's open-ended level scale.
	SlogLevelTrace    slog.Level = slog.LevelDebug - 4
	SlogLevelCritical slog.Level = slog.LevelError + 4
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

// SlogLevel maps the WasmEdge severity to slog's level scale.
func (l LogLevel) SlogLevel() slog.Level {
	switch l {
	case LogLevelTrace:
		return SlogLevelTrace
	case LogLevelDebug:
		return slog.LevelDebug
	case LogLevelInfo:
		return slog.LevelInfo
	case LogLevelWarn:
		return slog.LevelWarn
	case LogLevelError:
		return slog.LevelError
	case LogLevelCritical:
		return SlogLevelCritical
	default:
		return slog.LevelInfo
	}
}

// RouteLogsToSlog forwards engine logs to a slog.Logger, preserving the
// native timestamp, logger name, thread ID, and all six severity levels.
func RouteLogsToSlog(l *slog.Logger) {
	if l == nil {
		SetLogCallback(nil)
		return
	}
	SetLogCallback(func(m LogMessage) {
		ctx := context.Background()
		handler := l.Handler()
		level := m.Level.SlogLevel()
		if !handler.Enabled(ctx, level) {
			return
		}
		record := slog.NewRecord(m.Time, level, m.Message, 0)
		record.AddAttrs(
			slog.String("logger", m.LoggerName),
			slog.Uint64("thread_id", m.ThreadID),
		)
		_ = handler.Handle(ctx, record)
	})
}
