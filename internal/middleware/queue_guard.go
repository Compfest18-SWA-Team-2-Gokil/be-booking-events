package middleware

import (
	"encoding/json"
	"net/http"

	queueapp "github.com/ebk-tech/be-booking-events/internal/queue/application"
	queuedomain "github.com/ebk-tech/be-booking-events/internal/queue/domain"
)

// QueueTokenGuard memvalidasi X-Queue-Token jika header dikirim.
// Jika header ada tapi token invalid/expired → 403.
// Jika header tidak ada → lolos (event mungkin tidak dalam mode antrian).
//
// Bypass prevention penuh (cek apakah event sedang dalam mode antrian)
// dilakukan di handler CreateOrder/HoldTicket karena event_id ada di request body,
// bukan di URL param.
func QueueTokenGuard(validateUC *queueapp.ValidateQueueTokenUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Queue-Token")
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			if _, err := validateUC.Execute(r.Context(), token); err != nil {
				status := http.StatusForbidden
				msg := "queue token tidak valid"
				if err == queuedomain.ErrTokenExpired {
					msg = "queue token sudah kadaluarsa, silakan join antrian kembali"
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(map[string]string{"error": msg})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
