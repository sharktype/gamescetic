package routing

import (
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

// rateLimit throttles a single operation per client IP. It should be placed on an API group.
func rateLimit(limit int) func(huma.Context, func(huma.Context)) {
	limiter := httprate.NewRateLimiter(limit, rateLimitWindow)
	return func(ctx huma.Context, next func(huma.Context)) {
		request, writer := humachi.Unwrap(ctx)

		ip, err := httprate.KeyByIP(request)
		if err != nil {
			ip = request.RemoteAddr
		}

		key := ip + " " + ctx.Operation().OperationID
		if limiter.RespondOnLimit(writer, request, key) {
			return
		}

		next(ctx)
	}
}
