package httpx

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type StatusError struct {
	huma.ErrorModel
	cause error
}

func NewStatusError(status int, message string, cause error) *StatusError {
	return &StatusError{
		Status: status,
		Title:  http.StatusText(status),
		Detail: message,
		cause:  cause,
	}
}

func (e *StatusError) Unwrap() error {
	return e.cause
}
