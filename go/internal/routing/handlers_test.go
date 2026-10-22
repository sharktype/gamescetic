package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/sharktype/satchel/internal/auth/authtest"
	"github.com/sharktype/satchel/internal/clients"
	"github.com/sharktype/satchel/internal/db/dbtest"
	"github.com/sharktype/satchel/internal/httpx"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestHandle(t *testing.T) {
	t.Run("the specification documents every operation", func(t *testing.T) {
		router, _ := newHandledRouter(t, dbtest.NewClosed(t))
		response := get(router, "/openapi.json", "")

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		var spec struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
			t.Fatal(err)
		}

		for _, path := range []string{"/health", "/health/db", "/health/auth", "/health/error"} {
			if _, ok := spec.Paths[path]; !ok {
				t.Fatalf("the specification is missing %q", path)
			}
		}
	})

	t.Run("auth health answers with the user", func(t *testing.T) {
		user := dbtest.ProvisionedUser

		router, signer := newHandledRouter(t, dbtest.Provision(t))

		response := get(router, "/health/auth", "Bearer "+signer.Token(t, user.Sub))

		if response.Code != http.StatusOK {
			t.Fatalf("got %d, want %d", response.Code, http.StatusOK)
		}

		var body struct {
			Message string `json:"message"`
		}

		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}

		expected := fmt.Sprintf("%s (%s, %s) OK", user.ID, user.Name, user.Email)
		if body.Message != expected {
			t.Fatalf("got %q, want %q", body.Message, expected)
		}
	})

	t.Run("a wrapped status error logs its cause", func(t *testing.T) {
		cause := errors.New("dial tcp 10.0.0.1:5432: connect: connection refused")

		core, logs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(core)

		api := newTestAPI(t)

		register(logger, api.public, huma.Operation{
			OperationID: "GetWrappedError",
			Method:      http.MethodGet,
			Path:        "/wrapped-error",
		}, func(context.Context, *struct{}) (*struct{}, error) {
			return nil, fmt.Errorf("wrapped: %w", httpx.NewStatusError(
				http.StatusServiceUnavailable, "database unavailable", cause,
			))
		})

		response := get(api.router, "/wrapped-error", "")
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("got %d, want %d", response.Code, http.StatusServiceUnavailable)
		}

		entries := logs.FilterMessage("handler failed").All()
		if len(entries) != 1 {
			t.Fatalf("got %d matching log entries, want 1", len(entries))
		}

		if logged, _ := entries[0].ContextMap()["error"].(string); !strings.Contains(
			logged, "connection refused",
		) {
			t.Fatalf("the log entry lost the cause: %q", logged)
		}
	})

	t.Run("a status error without a cause is logged", func(t *testing.T) {
		core, logs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(core)

		api := newTestAPI(t)

		register(logger, api.public, huma.Operation{
			OperationID: "GetPlainError",
			Method:      http.MethodGet,
			Path:        "/plain-error",
		}, func(context.Context, *struct{}) (*struct{}, error) {
			return nil, huma.Error500InternalServerError("intentional failure")
		})

		response := get(api.router, "/plain-error", "")
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("got %d, want %d", response.Code, http.StatusInternalServerError)
		}

		entries := logs.FilterMessage("handler failed").All()
		if len(entries) != 1 {
			t.Fatalf("got %d matching log entries, want 1", len(entries))
		}

		if logged, _ := entries[0].ContextMap()["error"].(string); !strings.Contains(
			logged, "intentional failure",
		) {
			t.Fatalf("the log entry lost the error: %q", logged)
		}
	})

	t.Run("a handler returning no output still reports its status", func(t *testing.T) {
		router, signer := newHandledRouter(t, dbtest.Provision(t))

		code := get(
			router, "/health/error", "Bearer "+signer.Token(t, dbtest.ProvisionedUser.Sub),
		).Code
		if code != http.StatusInternalServerError {
			t.Fatalf("got %d, want %d", code, http.StatusInternalServerError)
		}
	})

	t.Run("each operation is limited on its own", func(t *testing.T) {
		if publicRateLimit >= globalRateLimit {
			t.Fatalf(
				"this test needs publicRateLimit (%d) below globalRateLimit (%d)",
				publicRateLimit, globalRateLimit,
			)
		}

		router, signer := newHandledRouter(t, dbtest.Provision(t))

		for i := range publicRateLimit {
			if code := get(router, "/health", "").Code; code != http.StatusOK {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusOK)
			}
		}

		if code := get(router, "/health", "").Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}

		code := get(
			router, "/health/error", "Bearer "+signer.Token(t, dbtest.ProvisionedUser.Sub),
		).Code
		if code == http.StatusTooManyRequests {
			t.Fatal("exhausting /health also exhausted /health/error")
		}
	})

	t.Run("rejected requests still count against the endpoint", func(t *testing.T) {
		if securedRateLimit >= globalRateLimit {
			t.Fatalf(
				"this test needs securedRateLimit (%d) below globalRateLimit (%d)",
				securedRateLimit, globalRateLimit,
			)
		}

		router, _ := newHandledRouter(t, dbtest.NewClosed(t))

		for i := range securedRateLimit {
			if code := get(
				router, "/health/auth", "Bearer not-a-token",
			).Code; code != http.StatusUnauthorized {
				t.Fatalf("request %d: got %d, want %d", i+1, code, http.StatusUnauthorized)
			}
		}

		if code := get(
			router, "/health/auth", "Bearer not-a-token",
		).Code; code != http.StatusTooManyRequests {
			t.Fatalf("got %d, want %d", code, http.StatusTooManyRequests)
		}
	})

	t.Run("a failing dependency logs its cause", func(t *testing.T) {
		core, logs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(core)

		unavailable := dbtest.NewClosed(t)

		cause := unavailable.Ping(t.Context())
		if cause == nil {
			t.Fatal("expected the closed database to fail")
		}

		router := testRouterWithLogger(t, logger)
		authClient := authtest.New(t, dbtest.Provision(t))

		Handle(
			logger,
			router,
			&clients.Clients{DB: unavailable, Auth: authClient.Client},
			huma.DefaultConfig("test", "1"),
		)

		response := get(
			router, "/health/db", "Bearer "+authClient.Token(t, dbtest.ProvisionedUser.Sub),
		)
		if response.Code != http.StatusServiceUnavailable {
			t.Fatalf("got %d, want %d", response.Code, http.StatusServiceUnavailable)
		}

		if strings.Contains(response.Body.String(), cause.Error()) {
			t.Fatalf("the response leaks the cause: %q", response.Body.String())
		}

		entries := logs.FilterMessage("handler failed").All()
		if len(entries) != 1 {
			t.Fatalf("got %d matching log entries, want 1", len(entries))
		}

		if logged, _ := entries[0].ContextMap()["error"].(string); !strings.Contains(
			logged, cause.Error(),
		) {
			t.Fatalf("the log entry lost the cause: %q", logged)
		}
	})
}
