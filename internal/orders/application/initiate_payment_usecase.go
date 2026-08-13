package application

import (
	"context"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type InitiatePaymentUseCase struct {
	repo     OrderRepository
	provider PaymentProvider
}

func NewInitiatePaymentUseCase(repo OrderRepository, provider PaymentProvider) *InitiatePaymentUseCase {
	return &InitiatePaymentUseCase{repo: repo, provider: provider}
}

type InitiatePaymentOutput struct {
	PaymentID  string `json:"payment_id"`
	InvoiceURL string `json:"invoice_url"`
}

func (uc *InitiatePaymentUseCase) Execute(ctx context.Context, orderID, buyerID string) (*InitiatePaymentOutput, error) {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.BuyerID != buyerID {
		return nil, domain.ErrOrderNotFound
	}

	if order.Status != domain.OrderStatusPending {
		return nil, domain.ErrOrderNotPending
	}

	email, err := uc.repo.GetBuyerEmail(ctx, buyerID)
	if err != nil {
		return nil, err
	}

	result, err := uc.provider.CreateInvoice(ctx, CreateInvoiceInput{
		ExternalID:  order.ID,
		Amount:      order.TotalAmount,
		PayerEmail:  email,
		Description: fmt.Sprintf("Pembelian tiket event %s", order.EventID),
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat invoice: %w", err)
	}

	payment := &domain.Payment{
		OrderID:          order.ID,
		Amount:           order.TotalAmount,
		Status:           domain.PaymentStatusPending,
		XenditInvoiceID:  result.InvoiceID,
		XenditInvoiceURL: result.InvoiceURL,
	}
	if err := uc.repo.CreatePayment(ctx, payment); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusPaymentPending); err != nil {
		return nil, err
	}

	return &InitiatePaymentOutput{
		PaymentID:  payment.ID,
		InvoiceURL: result.InvoiceURL,
	}, nil
}
