package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type ConfirmPaymentUseCase struct {
	repo OrderRepository
}

func NewConfirmPaymentUseCase(repo OrderRepository) *ConfirmPaymentUseCase {
	return &ConfirmPaymentUseCase{repo: repo}
}

// ConfirmPaymentInput berisi data dari Xendit webhook payload.
type ConfirmPaymentInput struct {
	XenditInvoiceID string
	ExternalID      string // == order ID kita
	PaymentMethod   string
	Status          string // "PAID" | "EXPIRED"
}

func (uc *ConfirmPaymentUseCase) Execute(ctx context.Context, input ConfirmPaymentInput) error {
	order, err := uc.repo.GetOrder(ctx, input.ExternalID)
	if err != nil {
		return err
	}

	payment, err := uc.repo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}

	if input.Status == "PAID" {
		payment.Status = domain.PaymentStatusSuccess
		payment.PaymentMethod = input.PaymentMethod
		if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
			return err
		}
		// UpdateOrderStatus juga update ticket_units → CONFIRMED dalam satu tx
		return uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusPaid)
	}

	// EXPIRED atau status lain → cancel
	payment.Status = domain.PaymentStatusFailed
	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return err
	}
	return uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusCancelled)
}
