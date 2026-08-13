package application_test

import (
	"context"
	"testing"

	"github.com/ebk-tech/be-booking-events/internal/orders/application"
	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

func TestCreateOrderUseCase_Success(t *testing.T) {
	repo := newFakeOrderRepo()
	uc := application.NewCreateOrderUseCase(repo)

	order, err := uc.Execute(context.Background(), application.CreateOrderInput{
		BuyerID: "buyer-1",
		EventID: "event-1",
		UnitIDs: []string{"unit-1", "unit-2"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if order.Status != domain.OrderStatusPending {
		t.Errorf("status = %s, want PENDING", order.Status)
	}
	if order.TotalAmount != 300000 {
		t.Errorf("total_amount = %d, want 300000", order.TotalAmount)
	}
}

func TestCreateOrderUseCase_NoUnits(t *testing.T) {
	uc := application.NewCreateOrderUseCase(newFakeOrderRepo())

	_, err := uc.Execute(context.Background(), application.CreateOrderInput{
		BuyerID: "buyer-1",
		EventID: "event-1",
		UnitIDs: []string{},
	})

	if err == nil {
		t.Fatal("expected error karena unit_ids kosong")
	}
}

func TestInitiatePaymentUseCase_Success(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID:          "order-1",
		BuyerID:     "buyer-1",
		Status:      domain.OrderStatusPending,
		TotalAmount: 150000,
	}
	repo.emails["buyer-1"] = "buyer@test.com"

	uc := application.NewInitiatePaymentUseCase(repo, &fakePaymentProvider{})
	out, err := uc.Execute(context.Background(), "order-1", "buyer-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.InvoiceURL == "" {
		t.Error("invoice_url kosong")
	}
	if repo.orders["order-1"].Status != domain.OrderStatusPaymentPending {
		t.Errorf("status = %s, want PAYMENT_PENDING", repo.orders["order-1"].Status)
	}
}

func TestInitiatePaymentUseCase_WrongBuyer(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID: "order-1", BuyerID: "buyer-1", Status: domain.OrderStatusPending,
	}

	uc := application.NewInitiatePaymentUseCase(repo, &fakePaymentProvider{})
	_, err := uc.Execute(context.Background(), "order-1", "buyer-lain")

	if err != domain.ErrOrderNotFound {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
}

func TestConfirmPaymentUseCase_Paid(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID: "order-1", BuyerID: "buyer-1", Status: domain.OrderStatusPaymentPending,
	}
	repo.payments["order-1"] = &domain.Payment{
		ID: "pay-1", OrderID: "order-1", XenditInvoiceID: "xendit-inv-123",
	}

	uc := application.NewConfirmPaymentUseCase(repo)
	err := uc.Execute(context.Background(), application.ConfirmPaymentInput{
		XenditInvoiceID: "xendit-inv-123",
		ExternalID:      "order-1",
		PaymentMethod:   "CREDIT_CARD",
		Status:          "PAID",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.orders["order-1"].Status != domain.OrderStatusPaid {
		t.Errorf("status = %s, want PAID", repo.orders["order-1"].Status)
	}
	if repo.payments["order-1"].Status != domain.PaymentStatusSuccess {
		t.Errorf("payment status = %s, want SUCCESS", repo.payments["order-1"].Status)
	}
}

func TestRequestRefundUseCase_Success(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID: "order-1", BuyerID: "buyer-1", Status: domain.OrderStatusPaid,
	}

	uc := application.NewRequestRefundUseCase(repo)
	err := uc.Execute(context.Background(), "order-1", "buyer-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.orders["order-1"].Status != domain.OrderStatusRefundRequested {
		t.Errorf("status = %s, want REFUND_REQUESTED", repo.orders["order-1"].Status)
	}
}

func TestRequestRefundUseCase_NotPaid(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID: "order-1", BuyerID: "buyer-1", Status: domain.OrderStatusPending,
	}

	uc := application.NewRequestRefundUseCase(repo)
	err := uc.Execute(context.Background(), "order-1", "buyer-1")

	if err != domain.ErrOrderNotPaid {
		t.Fatalf("expected ErrOrderNotPaid, got %v", err)
	}
}

func TestApproveRefundUseCase_Success(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID: "order-1", Status: domain.OrderStatusRefundRequested, TotalAmount: 150000,
	}
	repo.payments["order-1"] = &domain.Payment{
		ID: "pay-1", OrderID: "order-1", XenditInvoiceID: "xendit-inv-123", Amount: 150000,
	}

	uc := application.NewApproveRefundUseCase(repo, &fakePaymentProvider{})
	err := uc.Execute(context.Background(), "order-1")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.orders["order-1"].Status != domain.OrderStatusRefunded {
		t.Errorf("status = %s, want REFUNDED", repo.orders["order-1"].Status)
	}
	if repo.payments["order-1"].XenditRefundID == "" {
		t.Error("xendit_refund_id tidak tersimpan")
	}
}

func TestApproveRefundUseCase_NotRequested(t *testing.T) {
	repo := newFakeOrderRepo()
	repo.orders["order-1"] = &domain.Order{
		ID: "order-1", Status: domain.OrderStatusPaid,
	}

	uc := application.NewApproveRefundUseCase(repo, &fakePaymentProvider{})
	err := uc.Execute(context.Background(), "order-1")

	if err != domain.ErrRefundNotRequested {
		t.Fatalf("expected ErrRefundNotRequested, got %v", err)
	}
}
