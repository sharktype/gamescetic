package routing

import (
	"net"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

const (
	globalRateLimit = 120
	rateLimitWindow = time.Minute
)

// rateLimitGlobally throttles every request from one client IP.
//
// This is the volumetric backstop and must be less restictive than the operation-level limits.
func rateLimitGlobally(next http.Handler) http.Handler {
	key := func(request *http.Request) (string, error) {
		return clientIP(request), nil
	}

	return httprate.Limit(globalRateLimit, rateLimitWindow, httprate.WithKeyFuncs(key))(next)
}

// rateLimit throttles each operation per client IP.
func rateLimit(limit int) func(huma.Context, func(huma.Context)) {
	limiter := httprate.NewRateLimiter(limit, rateLimitWindow)
	return func(ctx huma.Context, next func(huma.Context)) {
		request, writer := humachi.Unwrap(ctx)

		key := clientIP(request) + " " + ctx.Operation().OperationID
		if limiter.RespondOnLimit(writer, request, key) {
			return
		}

		next(ctx)
	}
}

// clientIP returns the client's IP address.
//
// It prefers the address established by ClientIPFromXFF and falls back to the peer address for
// direct connections.
//
// Anything that is not an IP shares one bucket instead of getting one each, but this is just
// defensive, honestly.
func clientIP(request *http.Request) string {
	if ip := middleware.GetClientIP(request.Context()); ip != "" {
		return ip
	}

	ip, err := httprate.KeyByIP(request)
	if err != nil || net.ParseIP(ip) == nil {
		return "unknown"
	}

	return ip
}

// realIP copies the address captured at the edge onto RemoteAddr.
//
// This way, logs and other consumers do not report the proxy instead. Rate limits read the context
// value directly. Still works for self-hosting through something like direct Docker local
// networking via raw HTTP.
func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := middleware.GetClientIP(r.Context()); ip != "" {
			r.RemoteAddr = ip
		}

		next.ServeHTTP(w, r)
	})
}
