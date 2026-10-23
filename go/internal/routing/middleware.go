package routing

import (
	"net"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
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

// rateLimit throttles each operation independently per client IP.
func rateLimit(limit int) func(huma.Context, func(huma.Context)) {
	limiter := httprate.NewRateLimiter(limit, rateLimitWindow)
	return func(ctx huma.Context, next func(huma.Context)) {
		request, writer := humachi.Unwrap(ctx)

		// KeyByIP falls back to the raw RemoteAddr when it cannot split a host and port. If that
		// value is not an IP, using it as a key would give every connection its own "bucket." In
		// that case, we must share a single bucket for all connections.

		ip, err := httprate.KeyByIP(request)
		if err != nil || net.ParseIP(ip) == nil {
			ip = "unknown"
		}

		key := ip + " " + ctx.Operation().OperationID
		if limiter.RespondOnLimit(writer, request, key) {
			return
		}

		next(ctx)
	}
}
