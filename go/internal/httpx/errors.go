package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// StatusError is a huma.StatusError that carries a cause without exposing it in the response.
type StatusError struct {
	model *huma.ErrorModel
	cause error
}

func NewStatusError(status int, message string, cause error) *StatusError {
	return &StatusError{
		model: &huma.ErrorModel{
			Status: status,
			Title:  http.StatusText(status),
			Detail: message,
		},
		cause: cause,
	}
}

func (e *StatusError) Error() string {
	return e.model.Error()
}

func (e *StatusError) GetStatus() int {
	return e.model.GetStatus()
}

func (e *StatusError) ContentType(ct string) string {
	return e.model.ContentType(ct)
}

func (e *StatusError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.model)
}

func (e *StatusError) Unwrap() error {
	return e.cause
}
