package server

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"github.com/moensch/buildkite-github-token-server/api"
	"github.com/moensch/buildkite-github-token-server/internal/contextvalues"
)

// handleError provides a uniform way to emit errors out of our handlers. You should ALWAYS call
// return after calling it.
//
// err: if err is type api.HTTPError then it will get JSON-marshalled and returned to the user, this is for public consumption
func (srv *Server) handleError(w http.ResponseWriter, r *http.Request, err error, msg string, code int) {
	w.WriteHeader(code)

	httpErr, isHTTPError := err.(*api.HTTPError)
	if !isHTTPError {
		httpErr = &api.HTTPError{
			Message:   msg,
			RequestID: contextvalues.GetRequestID(r.Context()),
		}
	}

	bytes, marshalErr := json.Marshal(httpErr)
	if marshalErr != nil {
		srv.log.Error("failed to encode http error response")
	}
	_, _ = w.Write(bytes)

	fields := make([]zap.Field, 0)
	if msg != "" {
		fields = append(fields, zap.String("message", msg))
	}
	if err != nil {
		fields = append(fields, zap.String("error", err.Error()))
	}

	contextvalues.GetLogger(r.Context()).Error("error", fields...)
}
