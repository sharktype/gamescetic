package auth

import (
	"context"

	"github.com/sharktype/satchel/generated"
)

type contextKey int

const userKey contextKey = iota

// UserFromContext returns the user the middleware resolved for the request, if any.
func UserFromContext(ctx context.Context) (generated.User, bool) {
	user, ok := ctx.Value(userKey).(generated.User)

	return user, ok
}
