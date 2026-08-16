package application_test

import (
	"context"
	"errors"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type fakeOrderRepo struct {
	orders   map[string]*domain.Order
	payments map[string]*domain.Payment // keyed by order_id
	emails   map[string]string          // buyer_id → email
}

func newFakeOrderRepo() *fakeOrderRepo {
	return &fakeOrderRepo{
		orders:   make(map[string]*domain.Order),
		payments: make(map[string]*domain.Payment),
		emails:   make(map[string]string),
	}
}

func (r *fakeOrderRepo) CreateOrder(_ context.Context, buyerID, eventID string, unitIDs []string) (*domain.Order, error) {
	if len(unitIDs) == 0 {
		return nil, domain.ErrNoHeldUnits
	}
	o := &domain.Order{
		ID:          "order-" + buyerID,
		BuyerID:     buyerID,
		EventID:     eventID,
		Status:      domain.OrderStatusPending,
		TotalAmount: int64(len(unitIDs)) * 150000,
	}
	r.orders[o.ID] = o
	return o, nil
}

func (r *fakeOrderRepo) GetOrder(_ context.Context, orderID string) (*domain.Order, error) {
	o, ok := r.orders[orderID]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (r *fakeOrderRepo) UpdateOrderStatus(_ context.Context, orderID string, status domain.OrderStatus) error {
	o, ok := r.orders[orderID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.Status = status
	return nil
}

func (r *fakeOrderRepo) CreatePayment(_ context.Context, p *domain.Payment) error {
	p.ID = "pay-" + p.OrderID
	r.payments[p.OrderID] = p
	return nil
}

func (r *fakeOrderRepo) GetPaymentByOrderID(_ context.Context, orderID string) (*domain.Payment, error) {
	p, ok := r.payments[orderID]
	if !ok {
		return nil, domain.ErrPaymentNotFound
	}
	return p, nil
}

func (r *fakeOrderRepo) UpdatePayment(_ context.Context, p *domain.Payment) error {
	r.payments[p.OrderID] = p
	return nil
}

func (r *fakeOrderRepo) GetBuyerEmail(_ context.Context, buyerID string) (string, error) {
	email, ok := r.emails[buyerID]
	if !ok {
		return "", errors.New("user tidak ditemukan")
	}
	return email, nil
}

func (r *fakeOrderRepo) ConfirmOrderPayment(_ context.Context, orderID string) error {
	o, ok := r.orders[orderID]
	if !ok {
		return domain.ErrOrderNotFound
	}
	o.Status = domain.OrderStatusPaid
	return nil
}

func (r *fakeOrderRepo) HasAdmittedUnits(_ context.Context, _ string) (bool, error) {
	return false, nil
}

// fakePaymentProvider
type fakePaymentProvider struct {
	shouldFail bool
}

func (p *fakePaymentProvider) CreateInvoice(_ context.Context, input application.CreateInvoiceInput) (*application.InvoiceResult, error) {
	if p.shouldFail {
		return nil, errors.New("xendit error")
	}
	return &application.InvoiceResult{
		InvoiceID:  "xendit-inv-123",
		InvoiceURL: "https://checkout.xendit.co/pay/xendit-inv-123",
	}, nil
}

func (p *fakePaymentProvider) RefundPayment(_ context.Context, _ string, _ int64) (string, error) {
	if p.shouldFail {
		return "", errors.New("xendit refund error")
	}
	return "xendit-refund-456", nil
}
