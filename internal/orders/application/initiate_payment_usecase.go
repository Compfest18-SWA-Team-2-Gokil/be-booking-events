package application

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type InitiatePaymentUseCase struct {
	repo            OrderRepository
	provider        PaymentProvider
	frontendBaseURL string
}

func NewInitiatePaymentUseCase(repo OrderRepository, provider PaymentProvider, frontendBaseURL string) *InitiatePaymentUseCase {
	baseURL := frontendBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:5173"
	}
	return &InitiatePaymentUseCase{repo: repo, provider: provider, frontendBaseURL: baseURL}
}

type InitiatePaymentOutput struct {
	PaymentID       string `json:"payment_id"`
	XenditInvoiceID string `json:"xendit_invoice_id"`
	InvoiceURL      string `json:"invoice_url"`
}

func (uc *InitiatePaymentUseCase) Execute(ctx context.Context, orderID, buyerID string) (*InitiatePaymentOutput, error) {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if order.BuyerID != buyerID {
		return nil, domain.ErrOrderNotFound
	}

	if order.Status == domain.OrderStatusCancelled {
		return nil, domain.ErrOrderCancelled
	}

	// Jika order sudah PAYMENT_PENDING, gunakan kembali invoice URL yang sudah aktif
	if order.Status == domain.OrderStatusPaymentPending {
		payment, err := uc.repo.GetPaymentByOrderID(ctx, order.ID)
		if err == nil && payment != nil && payment.XenditInvoiceURL != "" {
			return &InitiatePaymentOutput{
				PaymentID:       payment.ID,
				XenditInvoiceID: payment.XenditInvoiceID,
				InvoiceURL:      payment.XenditInvoiceURL,
			}, nil
		}
	}

	if order.Status != domain.OrderStatusPending && order.Status != domain.OrderStatusPaymentPending {
		return nil, domain.ErrOrderNotPending
	}

	email, err := uc.repo.GetBuyerEmail(ctx, buyerID)
	if err != nil {
		return nil, err
	}

	result, err := uc.provider.CreateInvoice(ctx, CreateInvoiceInput{
		ExternalID:         order.ID,
		Amount:             order.TotalAmount,
		PayerEmail:         email,
		Description:        fmt.Sprintf("Pembelian tiket event %s", order.EventID),
		SuccessRedirectURL: fmt.Sprintf("%s/payment/callback?order_id=%s", uc.frontendBaseURL, order.ID),
		FailureRedirectURL: fmt.Sprintf("%s/payment/callback?order_id=%s", uc.frontendBaseURL, order.ID),
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
		PaymentID:       payment.ID,
		XenditInvoiceID: result.InvoiceID,
		InvoiceURL:      result.InvoiceURL,
	}, nil
}
