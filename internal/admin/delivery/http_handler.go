package delivery

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ebk-tech/be-booking-events/internal/admin/application"
	auditdomain "github.com/ebk-tech/be-booking-events/internal/audit/domain"
	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	ordersdomain "github.com/ebk-tech/be-booking-events/internal/orders/domain"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	adminUC *application.AdminUseCases
}

func NewAdminHandler(adminUC *application.AdminUseCases) *AdminHandler {
	return &AdminHandler{adminUC: adminUC}
}

// GET /api/v1/admin/disputes
func (h *AdminHandler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	orders, err := h.adminUC.ListDisputes(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memuat daftar sengketa/dispute")
		return
	}
	if orders == nil {
		orders = []*ordersdomain.Order{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"disputes": orders, "total": len(orders)})
}

type overrideRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// POST /api/v1/admin/orders/{orderID}/override
func (h *AdminHandler) OverrideOrderStatus(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	actorID := authdelivery.UserIDFromCtx(r.Context())

	var req overrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		writeError(w, http.StatusBadRequest, "status dan reason wajib diisi")
		return
	}

	err := h.adminUC.OverrideOrderStatus(r.Context(), application.OverrideOrderInput{
		OrderID:   orderID,
		NewStatus: ordersdomain.OrderStatus(req.Status),
		Reason:    req.Reason,
		ActorID:   actorID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "order status berhasil di-override", "status": req.Status})
}

type reassignRequest struct {
	TargetOrderID string `json:"target_order_id"`
	NewSeatNumber string `json:"new_seat_number"`
	Reason        string `json:"reason"`
}

// POST /api/v1/admin/tickets/{unitID}/reassign
func (h *AdminHandler) ReassignTicket(w http.ResponseWriter, r *http.Request) {
	unitID := chi.URLParam(r, "unitID")
	actorID := authdelivery.UserIDFromCtx(r.Context())

	var req reassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TargetOrderID == "" {
		writeError(w, http.StatusBadRequest, "target_order_id wajib diisi")
		return
	}

	err := h.adminUC.ReassignTicket(r.Context(), application.ReassignTicketInput{
		UnitID:        unitID,
		TargetOrderID: req.TargetOrderID,
		NewSeatNumber: req.NewSeatNumber,
		Reason:        req.Reason,
		ActorID:       actorID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "tiket berhasil dipindahtangankan"})
}

// GET /api/v1/admin/audit-logs
func (h *AdminHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	logs, err := h.adminUC.ListAuditLogs(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memuat audit logs")
		return
	}
	if logs == nil {
		logs = []*auditdomain.AuditLog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"audit_logs": logs, "total": len(logs)})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
