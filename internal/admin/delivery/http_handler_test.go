package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	adminapp "github.com/ebk-tech/be-booking-events/internal/admin/application"
	admindelivery "github.com/ebk-tech/be-booking-events/internal/admin/delivery"
	auditdomain "github.com/ebk-tech/be-booking-events/internal/audit/domain"
	ordersdomain "github.com/ebk-tech/be-booking-events/internal/orders/domain"
	"github.com/go-chi/chi/v5"
)

type fakeAdminRepo struct {
	disputes  []*ordersdomain.Order
	auditLogs []*auditdomain.AuditLog
}

func (r *fakeAdminRepo) ListDisputedOrders(ctx context.Context) ([]*ordersdomain.Order, error) {
	return r.disputes, nil
}

func (r *fakeAdminRepo) OverrideOrderStatus(ctx context.Context, orderID string, newStatus ordersdomain.OrderStatus, reason string, actorID string) error {
	r.auditLogs = append(r.auditLogs, &auditdomain.AuditLog{
		EntityName: "orders",
		EntityID:   orderID,
		Action:     "ADMIN_OVERRIDE",
		ToState:    string(newStatus),
	})
	return nil
}

func (r *fakeAdminRepo) ReassignTicket(ctx context.Context, unitID string, targetOrderID string, newSeatNumber string, reason string, actorID string) error {
	r.auditLogs = append(r.auditLogs, &auditdomain.AuditLog{
		EntityName: "ticket_units",
		EntityID:   unitID,
		Action:     "TICKET_REASSIGNMENT",
	})
	return nil
}

func (r *fakeAdminRepo) ListAuditLogs(ctx context.Context, limit int) ([]*auditdomain.AuditLog, error) {
	return r.auditLogs, nil
}

func TestAdminHandler(t *testing.T) {
	repo := &fakeAdminRepo{
		disputes: []*ordersdomain.Order{
			{ID: "order-1", Status: ordersdomain.OrderStatusPaymentDiscrepancy},
		},
		auditLogs: []*auditdomain.AuditLog{
			{EntityName: "orders", EntityID: "order-1", Action: "ADMIN_OVERRIDE"},
		},
	}
	uc := adminapp.NewAdminUseCases(repo)
	handler := admindelivery.NewAdminHandler(uc)

	r := chi.NewRouter()
	r.Get("/disputes", handler.ListDisputes)
	r.Post("/orders/{orderID}/override", handler.OverrideOrderStatus)
	r.Post("/tickets/{unitID}/reassign", handler.ReassignTicket)
	r.Get("/audit-logs", handler.ListAuditLogs)

	t.Run("List Disputes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/disputes", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("Override Order Status", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"status": "REFUNDED",
			"reason": "Customer payment discrepancy resolution",
		})
		req := httptest.NewRequest(http.MethodPost, "/orders/order-1/override", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("Reassign Ticket", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"target_order_id": "order-2",
			"new_seat_number": "A-12",
			"reason":          "Dispute resolution seat switch",
		})
		req := httptest.NewRequest(http.MethodPost, "/tickets/unit-1/reassign", bytes.NewReader(body))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})

	t.Run("List Audit Logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/audit-logs", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
	})
}
