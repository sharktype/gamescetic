package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sharktype/gamescetic/internal/auth"
	"github.com/sharktype/gamescetic/internal/db/dbtest"
)

func TestNewFailsWhenTheJWKSFetchFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("<html></html>"))
	}))
	defer server.Close()

	database := dbtest.NewClosed(t)
	result := make(chan error, 1)

	go func() {
		_, err := auth.New(context.Background(), database, auth.Config{
			JWKSURL:  server.URL,
			Issuer:   "issuer",
			Audience: "audience",
		})
		result <- err
	}()

	select {
	case err := <-result:
		if err == nil {
			t.Fatal("expected an error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("auth.New did not return: the startup JWKS fetch is unbounded")
	}
}
