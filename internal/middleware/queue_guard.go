package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	queueapp "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/application"
	queuedomain "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/domain"
)

// QueueTokenGuard memvalidasi X-Queue-Token dan mencegah bypass waiting-room.
//
// Logika:
//  1. Baca event_id dari request body (dan kembalikan body supaya handler bisa baca ulang).
//  2. Cek apakah event tersebut sedang dalam mode waiting-room (ada di active queues).
//  3. Jika event dalam mode waiting-room DAN token tidak dikirim → 403 (bypass dicegah).
//  4. Jika token dikirim, validasi token; kalau invalid/expired → 403.
//  5. Jika event tidak dalam mode waiting-room → lolos tanpa perlu token.
func QueueTokenGuard(validateUC *queueapp.ValidateQueueTokenUseCase, queueRepo queueapp.QueueRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Queue-Token")

			// Baca event_id dari body (non-destructive — restore body setelah baca)
			var eventID string
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(r.Body)
				r.Body.Close()
				// Restore body agar handler berikutnya masih bisa membaca
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

				if err == nil && len(bodyBytes) > 0 {
					var bodyPeek struct {
						EventID string `json:"event_id"`
					}
					_ = json.Unmarshal(bodyBytes, &bodyPeek)
					eventID = bodyPeek.EventID
				}
			}

			// Cek apakah event sedang dalam mode waiting-room
			inWaitingRoom := false
			if eventID != "" {
				size, err := queueRepo.QueueSize(r.Context(), eventID)
				if err == nil && size > 0 {
					inWaitingRoom = true
				}
			}

			if token == "" {
				if inWaitingRoom {
					// Fix #4: Event dalam mode waiting-room tapi tidak ada token → bypass dicegah
					respondJSON(w, http.StatusForbidden, "event sedang dalam mode antrean, sertakan X-Queue-Token dari antrean")
					return
				}
				// Event tidak dalam mode waiting-room → lolos
				next.ServeHTTP(w, r)
				return
			}

			// Token ada → validasi
			if _, err := validateUC.Execute(r.Context(), token); err != nil {
				status := http.StatusForbidden
				msg := "queue token tidak valid"
				if err == queuedomain.ErrTokenExpired {
					msg = "queue token sudah kadaluarsa, silakan join antrian kembali"
				}
				respondJSON(w, status, msg)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func respondJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
