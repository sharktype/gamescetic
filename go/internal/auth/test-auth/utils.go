package test_auth

import "github.com/sharktype/satchel/internal/auth"

func RequireJWT(scopes ...string) []map[string][]string {
	return []map[string][]string{{auth.DefaultJWTSchemeKey: scopes}}
}
