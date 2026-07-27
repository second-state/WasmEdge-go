package wasmedge

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type captureLogHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureLogHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureLogHandler) Handle(_ context.Context, record slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, record.Clone())
	return nil
}

func (h *captureLogHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *captureLogHandler) WithGroup(string) slog.Handler { return h }

func TestLogLevelSlogMapping(t *testing.T) {
	tests := []struct {
		in   LogLevel
		want slog.Level
	}{
		{LogLevelTrace, SlogLevelTrace},
		{LogLevelDebug, slog.LevelDebug},
		{LogLevelInfo, slog.LevelInfo},
		{LogLevelWarn, slog.LevelWarn},
		{LogLevelError, slog.LevelError},
		{LogLevelCritical, SlogLevelCritical},
		{LogLevel(99), slog.LevelInfo},
	}
	for _, test := range tests {
		if got := test.in.SlogLevel(); got != test.want {
			t.Errorf("%v.SlogLevel() = %v, want %v", test.in, got, test.want)
		}
	}
}

func TestRouteLogsToSlogPreservesRecord(t *testing.T) {
	handler := &captureLogHandler{}
	RouteLogsToSlog(slog.New(handler))
	t.Cleanup(func() { SetLogCallback(nil) })

	callback := logCallback.Load()
	if callback == nil {
		t.Fatal("RouteLogsToSlog did not install a callback")
	}
	when := time.Unix(1_700_000_000, 0)
	(*callback)(LogMessage{
		Message:    "compiled",
		LoggerName: "executor",
		Level:      LogLevelCritical,
		Time:       when,
		ThreadID:   42,
	})

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.records) != 1 {
		t.Fatalf("records = %d, want 1", len(handler.records))
	}
	record := handler.records[0]
	if record.Message != "compiled" || record.Level != SlogLevelCritical || !record.Time.Equal(when) {
		t.Fatalf("record = %+v", record)
	}
	attrs := map[string]any{}
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})
	if attrs["logger"] != "executor" || attrs["thread_id"] != uint64(42) {
		t.Fatalf("attrs = %#v", attrs)
	}
}
