package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/pabloadvisory/ssb-backend/internal/observability"
)

func Chain(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for index := len(middleware) - 1; index >= 0; index-- {
		handler = middleware[index](handler)
	}
	return handler
}

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("recovered HTTP panic", "panic", recovered, "request_id", RequestIDFromContext(request.Context()), "stack", string(debug.Stack()))
					WriteError(writer, request, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(writer, request)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(writer, request)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (recorder *responseRecorder) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *responseRecorder) Write(body []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}
	written, err := recorder.ResponseWriter.Write(body)
	recorder.bytes += written
	return written, err
}

func (recorder *responseRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

func AccessLog(logger *slog.Logger, metricAdapters ...observability.Metrics) func(http.Handler) http.Handler {
	var metrics observability.Metrics = observability.NopMetrics{}
	if len(metricAdapters) > 0 && metricAdapters[0] != nil {
		metrics = metricAdapters[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			recorder := &responseRecorder{ResponseWriter: writer}
			next.ServeHTTP(recorder, request)
			duration := time.Since(started)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			route := request.Pattern
			if route == "" {
				route = "unmatched"
			}
			metrics.ObserveHTTPRequest(request.Method, route, status, duration)
			logger.LogAttrs(request.Context(), slog.LevelInfo, "http request",
				slog.String("request_id", RequestIDFromContext(request.Context())),
				slog.String("method", request.Method),
				slog.String("path", request.URL.Path),
				slog.Int("status", status),
				slog.Int("response_bytes", recorder.bytes),
				slog.Duration("duration", duration),
				slog.String("client_ip", ClientIPFromContext(request.Context())),
				slog.String("remote_addr", request.RemoteAddr),
			)
		})
	}
}
