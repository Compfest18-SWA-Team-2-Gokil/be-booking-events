package middleware

import (
	"bytes"
	"net/http"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/appconfig"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/errorlog"
)

type errorLoggingResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *errorLoggingResponseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *errorLoggingResponseRecorder) Write(b []byte) (int, error) {
	if r.statusCode >= 500 {
		r.body.Write(b)
	}
	return r.ResponseWriter.Write(b)
}

func ErrorLogger(logger *errorlog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !appconfig.IsDebug() || logger == nil {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := errorlog.WithErrorSlot(r.Context())
			r = r.WithContext(ctx)

			rec := &errorLoggingResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rec, r)

			if rec.statusCode >= 500 {
				body := rec.body.String()
				if len(body) > 1024 {
					body = body[:1024]
				}
				actualErr := errorlog.GetError(r.Context())
				logger.LogEntry(r.Method, r.URL.Path, rec.statusCode, actualErr, body)
			}
		})
	}
}
