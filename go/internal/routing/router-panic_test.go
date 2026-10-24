package routing

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

type recordingTransport struct {
	mu     sync.Mutex
	events []*sentry.Event
}

func (t *recordingTransport) Configure(sentry.ClientOptions) {}

func (t *recordingTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.events = append(t.events, event)
}

func (t *recordingTransport) Flush(time.Duration) bool { return true }

func (t *recordingTransport) FlushWithContext(context.Context) bool { return true }

func (t *recordingTransport) Close() {}

func TestRecoverer(t *testing.T) {
	t.Run("panics are recovered", func(t *testing.T) {
		router := testRouter(t)

		router.Get("/panic", func(http.ResponseWriter, *http.Request) {
			panic("arbitrary panic")
		})

		if code := get(router, "/panic", "").Code; code != http.StatusInternalServerError {
			t.Fatalf("got %d, want %d", code, http.StatusInternalServerError)
		}
	})

	t.Run("panics are reported to the bugtracker", func(t *testing.T) {
		transport := &recordingTransport{}

		err := sentry.Init(sentry.ClientOptions{
			Dsn:       "https://public@example.com/1",
			Transport: transport,
		})
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() { sentry.CurrentHub().BindClient(nil) })

		router := testRouter(t)

		router.Get("/panic", func(http.ResponseWriter, *http.Request) {
			panic("arbitrary panic")
		})

		get(router, "/panic", "")
		sentry.Flush(time.Second)

		transport.mu.Lock()
		defer transport.mu.Unlock()

		if len(transport.events) != 1 {
			t.Fatalf("got %d events, want 1", len(transport.events))
		}

		event := transport.events[0]

		if event.Request == nil || event.Request.Method != http.MethodGet {
			t.Fatalf("got request %+v, want the panicking request", event.Request)
		}

		if event.Tags["request_id"] == "" {
			t.Fatal("the event lost the request ID")
		}

		exceptions := event.Exception
		if len(exceptions) != 1 || !strings.Contains(exceptions[0].Value, "arbitrary panic") {
			t.Fatalf("got exceptions %v, want the panic", exceptions)
		}

		if exceptions[0].Stacktrace == nil {
			t.Fatal("the event lost the stack trace")
		}
	})
}
