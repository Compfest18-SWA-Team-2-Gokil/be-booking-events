package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	"github.com/ebk-tech/be-booking-events/internal/admin/application"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	listDisputesUC  *application.ListDisputesUseCase
	overrideOrderUC *application.OverrideOrderUseCase
}

func NewAdminHandler(listUC *application.ListDisputesUseCase, overrideUC *application.OverrideOrderUseCase) *AdminHandler {
	return &AdminHandler{listDisputesUC: listUC, overrideOrderUC: overrideUC}
}

// GET /api/v1/admin/disputes
func (h *AdminHandler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	disputes, err := h.listDisputesUC.Execute(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data dispute")
		return
	}
	if disputes == nil {
		disputes = []application.DisputeOrder{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"disputes": disputes})
}

// POST /api/v1/admin/orders/{orderID}/override
func (h *AdminHandler) OverrideOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	adminID := authdelivery.UserIDFromCtx(r.Context())

	var req struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" || req.Reason == "" {
		writeError(w, http.StatusBadRequest, "field 'status' dan 'reason' wajib diisi")
		return
	}

	err := h.overrideOrderUC.Execute(r.Context(), orderID, adminID, req.Status, req.Reason)
	if err != nil {
		if errors.Is(err, application.ErrInvalidOverrideStatus) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal override order")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"order_id": orderID,
		"status":   req.Status,
		"message":  "order berhasil di-override dan tercatat di audit log",
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
