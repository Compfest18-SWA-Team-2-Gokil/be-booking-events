package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

const idempotencyTTL = 24 * time.Hour

type cachedResponse struct {
	StatusCode int               `json:"status_code"`
	Body       []byte            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

// Idempotency middleware mencegah duplikat request berdasarkan header Idempotency-Key.
// Respons pertama di-cache di Redis selama 24 jam.
// Request berikutnya dengan key yang sama langsung mendapat respons yang sama.
func Idempotency(rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			redisKey := "idempotency:" + r.Method + ":" + r.URL.Path + ":" + key
			ctx := context.Background()

			// Cek cache
			cached, err := rdb.Get(ctx, redisKey).Bytes()
			if err == nil {
				var resp cachedResponse
				if json.Unmarshal(cached, &resp) == nil {
					w.Header().Set("Idempotency-Replayed", "true")
					for k, v := range resp.Headers {
						w.Header().Set(k, v)
					}
					w.WriteHeader(resp.StatusCode)
					w.Write(resp.Body)
					return
				}
			}

			// Rekam respons
			rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rec, r)

			// Hanya cache respons sukses (2xx)
			if rec.statusCode >= 200 && rec.statusCode < 300 {
				resp := cachedResponse{
					StatusCode: rec.statusCode,
					Body:       rec.body.Bytes(),
					Headers:    map[string]string{"Content-Type": w.Header().Get("Content-Type")},
				}
				if data, err := json.Marshal(resp); err == nil {
					rdb.Set(ctx, redisKey, data, idempotencyTTL)
				}
			}
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
