package api

import "fmt"

func newHTTPError(message string) *HTTPError {
	return &HTTPError{
		Message: message,
	}
}

type HTTPError struct {
	Field     string `json:"field,omitempty"`
	Message   string `json:"message"`
	RequestID string `json:"req_id"`
}

func (h *HTTPError) withField(field string) *HTTPError {
	h.Field = field
	return h
}

func (h *HTTPError) Error() string {
	return fmt.Sprintf("%s: %s", h.Field, h.Message)
}
