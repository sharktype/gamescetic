package bugtracker

import "testing"

func TestInit(t *testing.T) {
	t.Run("an empty DSN disables the bugtracker", func(t *testing.T) {
		shutdown, err := Init("")
		if err != nil {
			t.Fatalf("got error %v, want none", err)
		}

		shutdown()
	})

	t.Run("an invalid DSN is rejected", func(t *testing.T) {
		shutdown, err := Init("not-a-dsn")
		if err == nil {
			t.Fatal("expected an error")
		}

		if shutdown != nil {
			t.Fatal("expected no shutdown function")
		}
	})
}
