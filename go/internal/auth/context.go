package auth

import "context"

type contextKey int

const userKey contextKey = iota

// UserFromContext returns the user the middleware resolved for the request, if there was one.
func UserFromContext(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userKey).(User)

	return user, ok
}
