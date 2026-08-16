package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	authdelivery "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/delivery"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/admin/application"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	listDisputesUC   *application.ListDisputesUseCase
	overrideOrderUC  *application.OverrideOrderUseCase
	listAuditLogsUC  *application.ListAuditLogsUseCase
}

func NewAdminHandler(
	listUC *application.ListDisputesUseCase,
	overrideUC *application.OverrideOrderUseCase,
	auditUC *application.ListAuditLogsUseCase,
) *AdminHandler {
	return &AdminHandler{
		listDisputesUC:  listUC,
		overrideOrderUC: overrideUC,
		listAuditLogsUC: auditUC,
	}
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
	writeJSON(w, http.StatusOK, map[string]any{
		"total":    len(disputes),
		"disputes": disputes,
	})
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

// GET /api/v1/admin/audit-logs
func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	logs, err := h.listAuditLogsUC.Execute(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data audit logs")
		return
	}
	if logs == nil {
		logs = []application.AuditLogEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total":      len(logs),
		"audit_logs": logs,
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
