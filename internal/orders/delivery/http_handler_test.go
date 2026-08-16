package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authdelivery "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/delivery"
	ordersapp "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/delivery"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
	"github.com/go-chi/chi/v5"
)

// === FAKE IMPLEMENTATIONS UNTUK TESTING HTTP HANDLERS ===

type fakeOrderRepository struct {
	orders       map[string]*domain.Order
	payments     map[string]*domain.Payment
	emails       map[string]string
	hasAdmitted  bool
	lostSeatFail bool
}

func newFakeOrderRepository() *fakeOrderRepository {
	return &fakeOrderRepository{
		orders:   make(map[string]*domain.Order),
		payments: make(map[string]*domain.Payment),
		emails:   make(map[string]string),
	}
}

func (r *fakeOrderRepository) CreateOrder(ctx context.Context, buyerID, eventID string, unitIDs []string) (*domain.Order, error) {
	if len(unitIDs) == 0 {
		return nil, domain.ErrNoHeldUnits
	}
	o := &domain.Order{
		ID:          "order-123",
		BuyerID:     buyerID,
		EventID:     eventID,
		Status:      domain.OrderStatusPending,
		TotalAmount: int64(len(unitIDs)) * 100000,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	r.orders[o.ID] = o
	return o, nil
}

func (r *fakeOrderRepository) GetOrder(ctx context.Context, orderID string) (*domain.Order, error) {
	o, ok := r.orders[orderID]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (r *fakeOrderRepository) GetOrdersByBuyer(_ context.Context, buyerID string, _, _ int) ([]*domain.Order, int, error) {
	var list []*domain.Order
	for _, o := range r.orders {
		if o.BuyerID == buyerID {
			list = append(list, o)
		}
	}
	return list, len(list), nil
}

func (r *fakeOrderRepository) UpdateOrderStatus(ctx context.Context, orderID string, status domain.OrderStatus) error {
	o, ok := r.orders[orderID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.Status = status
	return nil
}

func (r *fakeOrderRepository) ConfirmOrderPayment(ctx context.Context, orderID string) error {
	if r.lostSeatFail {
		return domain.ErrLostSeat
	}
	return r.UpdateOrderStatus(ctx, orderID, domain.OrderStatusPaid)
}

func (r *fakeOrderRepository) HasAdmittedUnits(ctx context.Context, orderID string) (bool, error) {
	return r.hasAdmitted, nil
}

func (r *fakeOrderRepository) CreatePayment(ctx context.Context, p *domain.Payment) error {
	p.ID = "pay-123"
	r.payments[p.OrderID] = p
	return nil
}

func (r *fakeOrderRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	p, ok := r.payments[orderID]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	return p, nil
}

func (r *fakeOrderRepository) UpdatePayment(ctx context.Context, p *domain.Payment) error {
	r.payments[p.OrderID] = p
	return nil
}

func (r *fakeOrderRepository) GetBuyerEmail(ctx context.Context, buyerID string) (string, error) {
	email, ok := r.emails[buyerID]
	if !ok {
		return "", errors.New("buyer tidak ditemukan")
	}
	return email, nil
}

func (r *fakeOrderRepository) GetEventDate(_ context.Context, _ string) (time.Time, error) {
	return time.Now().Add(48 * time.Hour), nil // Default H+2 (valid)
}

func (r *fakeOrderRepository) GetRefundRequestsByOrganizer(_ context.Context, _ string, _, _ int) ([]*ordersapp.RefundRequestItem, int, error) {
	return nil, 0, nil
}

type fakePaymentProvider struct {
	shouldFail bool
}

func (p *fakePaymentProvider) CreateInvoice(ctx context.Context, input ordersapp.CreateInvoiceInput) (*ordersapp.InvoiceResult, error) {
	if p.shouldFail {
		return nil, errors.New("xendit error")
	}
	return &ordersapp.InvoiceResult{
		InvoiceID:  "xendit-inv-abc",
		InvoiceURL: "https://checkout.xendit.co/pay/xendit-inv-abc",
	}, nil
}

func (p *fakePaymentProvider) RefundPayment(ctx context.Context, invoiceID string, amount int64) (string, error) {
	if p.shouldFail {
		return "", errors.New("xendit refund error")
	}
	return "xendit-ref-xyz", nil
}

type mockTokenProvider struct {
	VerifyFunc func(token string) (string, string, error)
}

func (m *mockTokenProvider) Generate(userID string, role string) (string, error) {
	return "mock-token", nil
}

func (m *mockTokenProvider) Verify(token string) (string, string, error) {
	if m.VerifyFunc != nil {
		return m.VerifyFunc(token)
	}
	return "buyer-123", "BUYER", nil
}

// === UNIT TESTS UNTUK ORDERS HTTP HANDLERS ===

func setupTestRouter(handler *delivery.OrdersHandler, tokenProvider *mockTokenProvider) http.Handler {
	r := chi.NewRouter()
	authMiddleware := authdelivery.AuthMiddleware(tokenProvider, nil)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/payments/webhook/xendit", handler.XenditWebhook)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/orders", handler.CreateOrder)
			r.Get("/orders/{orderID}", handler.GetOrder)
			r.Post("/orders/{orderID}/pay", handler.InitiatePayment)
			r.Post("/orders/{orderID}/refund", handler.RequestRefund)
			r.Post("/orders/{orderID}/refund/approve", handler.ApproveRefund)
		})
	})
	return r
}

func TestOrdersHandler_CreateOrder(t *testing.T) {
	repo := newFakeOrderRepository()
	createOrderUC := ordersapp.NewCreateOrderUseCase(repo)
	handler := delivery.NewOrdersHandler(createOrderUC, nil, nil, nil, nil, nil, repo, "")
	router := setupTestRouter(handler, &mockTokenProvider{})

	t.Run("Success", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"event_id":  "event-123",
			"unit_ids":  []string{"unit-1", "unit-2"},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
		}

		var respOrder domain.Order
		if err := json.Unmarshal(rec.Body.Bytes(), &respOrder); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if respOrder.ID != "order-123" {
			t.Errorf("order id = %s, want order-123", respOrder.ID)
		}
	})

	t.Run("Empty Units - Bad Request", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]any{
			"event_id":  "event-123",
			"unit_ids":  []string{},
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewReader(reqBody))
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestOrdersHandler_GetOrder(t *testing.T) {
	repo := newFakeOrderRepository()
	repo.orders["order-123"] = &domain.Order{
		ID:          "order-123",
		BuyerID:     "buyer-123",
		EventID:     "event-123",
		Status:      domain.OrderStatusPending,
		TotalAmount: 150000,
	}
	getOrderUC := ordersapp.NewGetOrderUseCase(repo)
	handler := delivery.NewOrdersHandler(nil, nil, nil, nil, nil, getOrderUC, repo, "")
	router := setupTestRouter(handler, &mockTokenProvider{})

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/order-123", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/order-999", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})
}

func TestOrdersHandler_InitiatePayment(t *testing.T) {
	repo := newFakeOrderRepository()
	repo.orders["order-123"] = &domain.Order{
		ID:          "order-123",
		BuyerID:     "buyer-123",
		EventID:     "event-123",
		Status:      domain.OrderStatusPending,
		TotalAmount: 150000,
	}
	repo.emails["buyer-123"] = "buyer@test.com"

	provider := &fakePaymentProvider{}
	initiatePayUC := ordersapp.NewInitiatePaymentUseCase(repo, provider)
	handler := delivery.NewOrdersHandler(nil, initiatePayUC, nil, nil, nil, nil, repo, "")
	router := setupTestRouter(handler, &mockTokenProvider{})

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/order-123/pay", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		var resp struct {
			PaymentID       string `json:"payment_id"`
			XenditInvoiceID string `json:"xendit_invoice_id"`
			InvoiceURL      string `json:"invoice_url"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.XenditInvoiceID != "xendit-inv-abc" {
			t.Errorf("invoice_id = %s, want xendit-inv-abc", resp.XenditInvoiceID)
		}
	})
}

func TestOrdersHandler_XenditWebhook(t *testing.T) {
	repo := newFakeOrderRepository()
	repo.orders["order-123"] = &domain.Order{
		ID:          "order-123",
		BuyerID:     "buyer-123",
		Status:      domain.OrderStatusPaymentPending,
		TotalAmount: 150000,
	}
	repo.payments["order-123"] = &domain.Payment{
		ID:              "pay-123",
		OrderID:         "order-123",
		XenditInvoiceID: "xendit-inv-abc",
		Status:          domain.PaymentStatusPending,
	}

	confirmPayUC := ordersapp.NewConfirmPaymentUseCase(repo, &fakePaymentProvider{})
	callbackToken := "supersecret"
	handler := delivery.NewOrdersHandler(nil, nil, confirmPayUC, nil, nil, nil, repo, callbackToken)
	router := setupTestRouter(handler, &mockTokenProvider{})

	t.Run("Success PAID", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]any{
			"id":             "xendit-inv-abc",
			"external_id":    "order-123",
			"status":         "PAID",
			"payment_method": "BANK_TRANSFER",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook/xendit", bytes.NewReader(payload))
		req.Header.Set("x-callback-token", callbackToken)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if repo.orders["order-123"].Status != domain.OrderStatusPaid {
			t.Errorf("order status = %s, want PAID", repo.orders["order-123"].Status)
		}
	})

	t.Run("Success EXPIRED", func(t *testing.T) {
		repo.orders["order-123"].Status = domain.OrderStatusPaymentPending
		payload, _ := json.Marshal(map[string]any{
			"id":             "xendit-inv-abc",
			"external_id":    "order-123",
			"status":         "EXPIRED",
			"payment_method": "",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook/xendit", bytes.NewReader(payload))
		req.Header.Set("x-callback-token", callbackToken)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if repo.orders["order-123"].Status != domain.OrderStatusCancelled {
			t.Errorf("order status = %s, want CANCELLED", repo.orders["order-123"].Status)
		}
	})

	t.Run("Unauthorized - Invalid Token", func(t *testing.T) {
		payload, _ := json.Marshal(map[string]any{
			"id":          "xendit-inv-abc",
			"external_id": "order-123",
			"status":      "PAID",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook/xendit", bytes.NewReader(payload))
		req.Header.Set("x-callback-token", "wrongtoken")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

func TestOrdersHandler_RequestRefund(t *testing.T) {
	repo := newFakeOrderRepository()
	repo.orders["order-123"] = &domain.Order{
		ID:          "order-123",
		BuyerID:     "buyer-123",
		Status:      domain.OrderStatusPaid,
		TotalAmount: 150000,
	}

	requestRefundUC := ordersapp.NewRequestRefundUseCase(repo)
	handler := delivery.NewOrdersHandler(nil, nil, nil, requestRefundUC, nil, nil, repo, "")
	router := setupTestRouter(handler, &mockTokenProvider{})

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/order-123/refund", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
}

func TestOrdersHandler_ApproveRefund(t *testing.T) {
	repo := newFakeOrderRepository()
	repo.orders["order-123"] = &domain.Order{
		ID:          "order-123",
		BuyerID:     "buyer-123",
		Status:      domain.OrderStatusRefundRequested,
		TotalAmount: 150000,
	}
	repo.payments["order-123"] = &domain.Payment{
		ID:              "pay-123",
		OrderID:         "order-123",
		XenditInvoiceID: "xendit-inv-abc",
		Amount:          150000,
		Status:          domain.PaymentStatusSuccess,
	}

	provider := &fakePaymentProvider{}
	approveRefundUC := ordersapp.NewApproveRefundUseCase(repo, provider)
	handler := delivery.NewOrdersHandler(nil, nil, nil, nil, approveRefundUC, nil, repo, "")
	
	// Gunakan role ORGANIZER agar lolos
	tokenProvider := &mockTokenProvider{
		VerifyFunc: func(token string) (string, string, error) {
			return "organizer-123", "ORGANIZER", nil
		},
	}
	router := setupTestRouter(handler, tokenProvider)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/order-123/refund/approve", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
	})
}
