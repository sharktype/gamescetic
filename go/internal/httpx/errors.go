package httpx

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// StatusError is a huma.StatusError that carries a cause without exposing it in the response.
type StatusError struct {
	*huma.ErrorModel
	cause error
}

func NewStatusError(status int, message string, cause error) *StatusError {
	return &StatusError{
		ErrorModel: &huma.ErrorModel{
			Status: status,
			Title:  http.StatusText(status),
			Detail: message,
		},
		cause: cause,
	}
}

func (e *StatusError) Error() string {
	return e.ErrorModel.Error()
}

func (e *StatusError) Unwrap() error {
	return e.cause
}
