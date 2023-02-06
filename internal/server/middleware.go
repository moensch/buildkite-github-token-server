package server

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	"github.com/moensch/buildkite-github-token-server/internal/contextvalues"
	"github.com/moensch/buildkite-github-token-server/internal/metrics"
)

func metricsMiddleware(endpoint string, next http.HandlerFunc) http.HandlerFunc {
	return promhttp.InstrumentHandlerDuration(metrics.PromMetrics.HTTPLatency.MustCurryWith(prometheus.Labels{"endpoint": endpoint}),
		promhttp.InstrumentHandlerCounter(metrics.PromMetrics.HTTPCount.MustCurryWith(prometheus.Labels{"handler": endpoint}),
			promhttp.InstrumentHandlerInFlight(metrics.PromMetrics.HTTPInflightRequestCount.With(prometheus.Labels{"endpoint": endpoint}), next),
		),
	)
}

func (srv *Server) jsonContentTypeMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// logMiddleware debug logs http requests
func (srv *Server) logMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		req_id := uuid.New()
		logger := srv.log.With(
			zap.String("req_id", req_id.String()),
		)

		ctx, cancel := context.WithTimeout(r.Context(), srv.config.ContextTimeout)
		ctx = contextvalues.SetLogger(ctx, logger)
		ctx = contextvalues.SetRequestID(ctx, req_id.String())
		r = r.WithContext(ctx)
		defer cancel()

		rec := writerRecorder{w, 200, []byte{}}
		next.ServeHTTP(&rec, r)

		logger.Info("http_request",
			zap.String("message", "http request finished"),
			zap.Int("status", rec.status),
			zap.String("request_uri", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("latency", fmt.Sprint(time.Since(start).Seconds())),
			zap.String("method", r.Method),
		)
	}
	return http.HandlerFunc(fn)
}

func (srv *Server) middlewareRecoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				var msg string
				switch v := rec.(type) {
				case error:
					msg = v.Error()
				case fmt.Stringer:
					msg = v.String()
				default:
					msg = fmt.Sprintf("%#v", rec)
				}

				logger := contextvalues.GetLogger(r.Context())
				logger.Error("error",
					zap.String("type", "panic"),
					zap.String("message", msg),
					zap.String("stack", string(debug.Stack())),
				)
				metrics.PromMetrics.PanicCount.Inc()

				// could cause a 'http: multiple response.WriteHeader calls' log if headers have already been set
				http.Error(w, fmt.Sprintf(`{"error": %q}`, http.StatusText(http.StatusInternalServerError)), http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// writerRecorder is a simple wrapper around the default http response writer to match its interface.
// it allows for the capture of the response body itself and the response code. This allows middlewares to capture the body and code.
type writerRecorder struct {
	http.ResponseWriter
	status int
	body   []byte
}

func (rec *writerRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *writerRecorder) Write(b []byte) (int, error) {
	dst := make([]byte, len(b))
	copy(dst, b)
	rec.body = dst
	return rec.ResponseWriter.Write(b)
}
