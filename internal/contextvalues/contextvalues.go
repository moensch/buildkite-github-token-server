package contextvalues

import (
	"context"
	"fmt"
	"runtime"

	"go.uber.org/zap"
)

type ctxkey string

const (
	ctxKeyLogger      ctxkey = "logger"
	ctxKeyRequestID   ctxkey = "req_id"
	ctxKeyLimitOffset ctxkey = "limitOffset"
	appName                  = "buildkite-github-token-server"
)

// GetLogger returns the ln logger out of the context
func GetLogger(ctx context.Context) *zap.Logger {
	logger := ctx.Value(ctxKeyLogger)
	if logger == nil {
		// If there is no logger, make a new one
		_, file, line, _ := runtime.Caller(2)
		// TODO What to do if this errors? panic()?
		zapLogger, _ := zap.NewDevelopment()
		NewLogger := zapLogger.With(zap.String("application", appName))
		// Log a unique error we can monitor our logs for and debug
		NewLogger.Error("nil logger instance requested",
			zap.String("caller", fmt.Sprintf("%v:%v", file, line)),
		)
		return NewLogger
	}
	return logger.(*zap.Logger)
}

// SetLogger sets the logger in the context
func SetLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, logger)
}

func GetRequestID(ctx context.Context) string {
	id := ctx.Value(ctxKeyRequestID)
	return id.(string)
}

func SetRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

type limitOffset struct {
	limit, offset int
}

func GetLimitOffset(ctx context.Context) (int, int) {
	limitAndOffset := ctx.Value(ctxKeyLimitOffset)
	limit := limitAndOffset.(*limitOffset).limit
	offset := limitAndOffset.(*limitOffset).offset
	return limit, offset
}

func SetLimitOffset(ctx context.Context, limit, offset int) context.Context {
	limitAndOffset := &limitOffset{
		limit:  limit,
		offset: offset,
	}
	return context.WithValue(ctx, ctxKeyLimitOffset, limitAndOffset)
}
