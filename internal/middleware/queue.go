package middleware

import (
	"encoding/json"
	"net/http"

	queueapp "github.com/ebk-tech/be-booking-events/internal/queue/application"
)

// RequireQueueToken memvalidasi header X-Queue-Token jika dikirimkan oleh klien.
// Jika queueUC nil, request diteruskan langsung.
func RequireQueueToken(validateUC *queueapp.ValidateQueueTokenUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := r.Header.Get("X-Queue-Token")
			// Jika client menyertakan X-Queue-Token, verifikasi keabsahannya
			if tokenStr != "" && validateUC != nil {
				_, err := validateUC.Execute(r.Context(), tokenStr)
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					_ = json.NewEncoder(w).Encode(map[string]string{
						"error": "queue token tidak valid atau sudah kadaluarsa",
					})
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
