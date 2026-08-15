package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
)

type ConfirmPaymentUseCase struct {
	repo     OrderRepository
	provider PaymentProvider
}

func NewConfirmPaymentUseCase(repo OrderRepository, provider PaymentProvider) *ConfirmPaymentUseCase {
	return &ConfirmPaymentUseCase{repo: repo, provider: provider}
}

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

	if input.Status != "PAID" {
		payment.Status = domain.PaymentStatusFailed
		if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
			return err
		}
		return uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusCancelled)
	}

	payment.Status = domain.PaymentStatusSuccess
	payment.PaymentMethod = input.PaymentMethod
	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return err
	}

	// ConfirmOrderPayment atomik: update ticket_units HELD→CONFIRMED, cek rows affected.
	// Jika 0 rows (tiket sudah direbut), order diset PAYMENT_DISCREPANCY dan return ErrLostSeat.
	if err := uc.repo.ConfirmOrderPayment(ctx, order.ID); err != nil {
		if errors.Is(err, domain.ErrLostSeat) {
			// Auto-refund: uang dikembalikan otomatis karena tiket tidak bisa dikonfirmasi.
			refundID, refundErr := uc.provider.RefundPayment(ctx, payment.XenditInvoiceID, payment.Amount)
			if refundErr == nil {
				payment.Status = domain.PaymentStatusRefunded
				payment.XenditRefundID = refundID
				_ = uc.repo.UpdatePayment(ctx, payment)
			}
			return fmt.Errorf("lost seat: %w", err)
		}
		return err
	}

	return nil
}
