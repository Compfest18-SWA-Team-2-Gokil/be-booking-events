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
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	disputes, total, err := h.listDisputesUC.Execute(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data dispute")
		return
	}
	if disputes == nil {
		disputes = []application.DisputeOrder{}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":    total,
		"disputes": disputes,
		"pagination": map[string]any{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
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
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	logs, total, err := h.listAuditLogsUC.Execute(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data audit logs")
		return
	}
	if logs == nil {
		logs = []application.AuditLogEntry{}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total":      total,
		"audit_logs": logs,
		"pagination": map[string]any{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
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
