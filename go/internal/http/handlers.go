package http

import (
	nethttp "net/http"

	"github.com/go-chi/cors"
	"github.com/sharktype/satchel/internal/utils"
)

type Origin string
type Method string

const (
	MethodGet     Method = nethttp.MethodGet
	MethodHead    Method = nethttp.MethodHead
	MethodPost    Method = nethttp.MethodPost
	MethodPut     Method = nethttp.MethodPut
	MethodPatch   Method = nethttp.MethodPatch
	MethodDelete  Method = nethttp.MethodDelete
	MethodOptions Method = nethttp.MethodOptions
)

type Header string

func MustNewCORSHandler(
	allowedOrigins []Origin,
	allowedMethods []Method,
	allowedHeaders, exposedHeaders []Header,
	allowCredentials bool,
	maxAge int,
) func(next nethttp.Handler) nethttp.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   utils.ToStrings(allowedOrigins),
		AllowedMethods:   utils.ToStrings(allowedMethods),
		AllowedHeaders:   utils.ToStrings(allowedHeaders),
		ExposedHeaders:   utils.ToStrings(exposedHeaders),
		AllowCredentials: allowCredentials,
		MaxAge:           maxAge,
	})
}
