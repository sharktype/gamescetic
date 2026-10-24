package logger

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestAppLog(t *testing.T) {
	t.Run("logs the message and fields", func(t *testing.T) {
		logs := replaceGlobalLogger(t)

		Info("hello", zap.String("key", "value"))

		entries := logs.All()
		if len(entries) != 1 {
			t.Fatalf("got %d log entries, want 1", len(entries))
		}

		entry := entries[0]
		if entry.Level != zapcore.InfoLevel {
			t.Fatalf("got level %v, want %v", entry.Level, zapcore.InfoLevel)
		}

		if entry.Message != "hello" {
			t.Fatalf("got message %q, want %q", entry.Message, "hello")
		}

		if got := entry.ContextMap()["key"]; got != "value" {
			t.Fatalf("got field %v, want %v", got, "value")
		}
	})

	t.Run("reports the call site", func(t *testing.T) {
		logs := replaceGlobalLogger(t)

		Info("hello")

		entries := logs.All()
		if len(entries) != 1 {
			t.Fatalf("got %d log entries, want 1", len(entries))
		}

		if caller := entries[0].Caller; !strings.HasSuffix(caller.File, "client_test.go") {
			t.Fatalf("got caller %+v, want client_test.go", caller)
		}
	})
}

func replaceGlobalLogger(t *testing.T) *observer.ObservedLogs {
	t.Helper()

	core, logs := observer.New(zapcore.InfoLevel)

	previous := zap.L()
	zap.ReplaceGlobals(zap.New(core, zap.AddCaller()))
	t.Cleanup(func() { zap.ReplaceGlobals(previous) })

	return logs
}
