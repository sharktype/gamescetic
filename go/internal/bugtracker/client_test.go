package bugtracker

import (
	"testing"

	"github.com/getsentry/sentry-go"
)

func TestInit(t *testing.T) {
	t.Run("a valid DSN in production enables the bugtracker", func(t *testing.T) {
		previous := sentry.CurrentHub().Client()
		t.Cleanup(func() { sentry.CurrentHub().BindClient(previous) })

		shutdown, err := Init("https://public@example.com/1", true)
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}
		t.Cleanup(shutdown)

		if !sentry.CurrentHub().Client().IsEnabled() {
			t.Fatal("the bugtracker was not initialized")
		}
	})

	t.Run("an empty DSN disables the bugtracker", func(t *testing.T) {
		shutdown, err := Init("", true)
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}

		shutdown()
	})

	t.Run("a DSN outside production is ignored", func(t *testing.T) {
		sentry.CurrentHub().BindClient(nil)

		shutdown, err := Init("https://public@example.com/1", false)
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}

		shutdown()

		if sentry.CurrentHub().Client().IsEnabled() {
			t.Fatal("the bugtracker was initialized outside production")
		}
	})

	t.Run("an invalid DSN is rejected", func(t *testing.T) {
		shutdown, err := Init("not-a-dsn", true)
		if err == nil {
			t.Fatal("expected an error")
		}

		if shutdown != nil {
			t.Fatal("expected no shutdown function")
		}
	})
}
