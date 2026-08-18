package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/domain"
	"github.com/go-chi/chi/v5"
)

const waitingRoomThreshold int64 = 100 // request/detik sebelum waiting room aktif

type QueueHandler struct {
	joinUC     *application.JoinQueueUseCase
	validateUC *application.ValidateQueueTokenUseCase
	repo       application.QueueRepository
}

func NewQueueHandler(
	joinUC *application.JoinQueueUseCase,
	validateUC *application.ValidateQueueTokenUseCase,
	repo application.QueueRepository,
) *QueueHandler {
	return &QueueHandler{joinUC: joinUC, validateUC: validateUC, repo: repo}
}

type joinRequest struct {
	UserID string `json:"user_id"`
}

type statusResponse struct {
	Status     string `json:"status"`               // "waiting" | "ready"
	Position   int64  `json:"position,omitempty"`
	QueueToken string `json:"queue_token,omitempty"`
}

// POST /api/v1/events/{eventID}/queue/join
// Entry point untuk waiting room. Cek rate terlebih dahulu — jika masih di bawah threshold,
// respons 200 tanpa antri. Jika melebihi threshold, masukkan ke antrean.
func (h *QueueHandler) JoinQueue(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")

	var req joinRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	rate, err := h.repo.IncrRequestRate(r.Context(), eventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	// Di bawah threshold: akses langsung tanpa antrean.
	if rate <= waitingRoomThreshold {
		writeJSON(w, http.StatusOK, map[string]string{"status": "direct"})
		return
	}

	out, err := h.joinUC.Execute(r.Context(), application.JoinQueueInput{
		EventID: eventID,
		UserID:  req.UserID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal masuk antrean")
		return
	}

	writeJSON(w, http.StatusAccepted, statusResponse{
		Status:   "waiting",
		Position: out.Position,
	})
}

// GET /api/v1/events/{eventID}/queue/status?user_id=xxx
// Polling endpoint: cek apakah user sudah mendapat queue token (gilirannya tiba).
func (h *QueueHandler) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id wajib diisi")
		return
	}

	token, err := h.repo.GetToken(r.Context(), eventID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if token != "" {
		writeJSON(w, http.StatusOK, statusResponse{Status: "ready", QueueToken: token})
		return
	}

	pos, err := h.repo.Position(r.Context(), eventID, userID)
	if err != nil || pos == -1 {
		writeError(w, http.StatusNotFound, "user tidak dalam antrean")
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{Status: "waiting", Position: pos})
}

// POST /api/v1/queue/token/validate
// Dipanggil sebagai middleware oleh modul checkout untuk memvalidasi queue token.
func (h *QueueHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.Header.Get("X-Queue-Token")
	if tokenStr == "" {
		writeError(w, http.StatusBadRequest, "X-Queue-Token header wajib diisi")
		return
	}

	token, err := h.validateUC.Execute(r.Context(), tokenStr)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "queue token sudah kadaluarsa")
		default:
			writeError(w, http.StatusUnauthorized, "queue token tidak valid")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"user_id":  token.UserID,
		"event_id": token.EventID,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
