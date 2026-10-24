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
	return httprate.LimitByIP(globalRateLimit, rateLimitWindow)(next)
}

// rateLimit throttles each operation per client IP.
func rateLimit(limit int) func(huma.Context, func(huma.Context)) {
	limiter := httprate.NewRateLimiter(limit, rateLimitWindow)
	return func(ctx huma.Context, next func(huma.Context)) {
		request, writer := humachi.Unwrap(ctx)

		ip, _ := httprate.KeyByIP(request)
		if net.ParseIP(ip) == nil {
			ip = "unknown"
		}

		key := ip + " " + ctx.Operation().OperationID
		if limiter.RespondOnLimit(writer, request, key) {
			return
		}

		next(ctx)
	}
}

// realIP copies the address captured by ClientIPFromXFF onto RemoteAddr.
//
// This gives logs and the rate limiters one source of truth. Requests without X-Forwarded-For
// keep their peer address. Still works for self-hosting through something like direct Docker
// local networking via raw HTTP.
func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := middleware.GetClientIP(r.Context()); ip != "" {
			r.RemoteAddr = ip
		}

		next.ServeHTTP(w, r)
	})
}
