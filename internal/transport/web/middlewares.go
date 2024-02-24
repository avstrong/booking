package web

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func (s *Server) loggerMiddleware() func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now().UTC()

			next.ServeHTTP(w, r)

			var traceID string

			if sc := trace.SpanContextFromContext(r.Context()); sc.IsValid() {
				traceID = sc.TraceID().String()
			}

			if spanTraceID := uuid.UUID(trace.SpanContextFromContext(r.Context()).TraceID()); spanTraceID != uuid.Nil {
				traceID = spanTraceID.String()
			}

			s.l.LogInfo(
				"type: access, method: %s, url: %s, proto: %s, userAgent: %s, traceID: %s, latency: %s",
				r.Method,
				r.URL.Path,
				r.Proto,
				r.Header.Get("User-Agent"),
				traceID,
				time.Since(start),
			)
		})
	}
}

func (s *Server) recoverMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if re := recover(); re != nil {
					err, ok := re.(error)
					if !ok {
						err = fmt.Errorf("%v: %w", re, ErrPanic)
					}
					s.l.LogErrorf("type: panic, error: %v\n", err)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) applyMiddlewares(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}

	return h
}
