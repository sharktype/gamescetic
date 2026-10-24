package bugtracker

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestAddSourceContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.go")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	eventWithFrame := func(frame sentry.Frame) *sentry.Event {
		return &sentry.Event{Exception: []sentry.Exception{{
			Stacktrace: &sentry.Stacktrace{Frames: []sentry.Frame{frame}},
		}}}
	}

	t.Run("a frame gets the lines around it", func(t *testing.T) {
		event := eventWithFrame(sentry.Frame{AbsPath: path, Lineno: 2})
		addSourceContext(event, nil)

		frame := event.Exception[0].Stacktrace.Frames[0]

		if frame.ContextLine != "two" {
			t.Fatalf("got context line %q, want %q", frame.ContextLine, "two")
		}

		if !slices.Equal(frame.PreContext, []string{"one"}) {
			t.Fatalf("got pre-context %q, want %q", frame.PreContext, []string{"one"})
		}

		if !slices.Equal(frame.PostContext, []string{"three"}) {
			t.Fatalf("got post-context %q, want %q", frame.PostContext, []string{"three"})
		}
	})

	t.Run("a missing source file is left alone", func(t *testing.T) {
		event := eventWithFrame(sentry.Frame{
			AbsPath: filepath.Join(t.TempDir(), "missing.go"),
			Lineno:  1,
		})
		addSourceContext(event, nil)

		frame := event.Exception[0].Stacktrace.Frames[0]
		if frame.ContextLine != "" {
			t.Fatalf("got context line %q, want none", frame.ContextLine)
		}
	})

	t.Run("a line past the end of the file is left alone", func(t *testing.T) {
		event := eventWithFrame(sentry.Frame{AbsPath: path, Lineno: 10})
		addSourceContext(event, nil)

		frame := event.Exception[0].Stacktrace.Frames[0]
		if frame.ContextLine != "" {
			t.Fatalf("got context line %q, want none", frame.ContextLine)
		}
	})

	t.Run("an event without a stack trace is left alone", func(t *testing.T) {
		event := &sentry.Event{Exception: []sentry.Exception{{}}}
		addSourceContext(event, nil)
	})
}
