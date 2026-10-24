package routing

import (
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

		key := ip + " " + ctx.Operation().OperationID
		if limiter.RespondOnLimit(writer, request, key) {
			return
		}

		next(ctx)
	}
}

// clientIP rewrites RemoteAddr to the rightmost X-Forwarded-For hop.
//
// We make the assumption that either the API sits behind exactly one proxy e.g. Cloudflare, or
// the service is deployed as a standalone server communicating over HTTP in client-trusted
// environments or in a private Docker network.
//
// The proxy is expected to append the client to X-Forwarded-For and cannot be directly modified by
// the client; if it is, only the rightmost hop is trusted so a prepended value is ignored/dropped
// and replaced with the actual client IP. Anything that doesn't append the XFF header must
// therefore be trusted or rate limits can be bypassed by a slightly crafty attacker. Requests
// without X-Forwarded-For keep their peer address.
//
// The rate limiters key on RemoteAddr through httprate.KeyByIP rather than on GetClientIP as
// KeyByIP collapses IPv6 to its /64; keying on the raw address would hand one IPv6 client 2^64
// buckets.
//
// As always, a reminder: production deployments should put a "real" rate limiter at the edge layer
// and not rely on this basic middleware implementation, especially when there are multiple
// replicas behind a load balancer.
func clientIP(next http.Handler) http.Handler {
	return middleware.ClientIPFromXFF()(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if ip := middleware.GetClientIP(r.Context()); ip != "" {
				r.RemoteAddr = ip
			}

			next.ServeHTTP(w, r)
		},
	))
}
