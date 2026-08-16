package application

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type ApproveRefundUseCase struct {
	repo     OrderRepository
	provider PaymentProvider
}

func NewApproveRefundUseCase(repo OrderRepository, provider PaymentProvider) *ApproveRefundUseCase {
	return &ApproveRefundUseCase{repo: repo, provider: provider}
}

func (uc *ApproveRefundUseCase) Execute(ctx context.Context, orderID string) error {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	if order.Status != domain.OrderStatusRefundRequested {
		return domain.ErrRefundNotRequested
	}

	payment, err := uc.repo.GetPaymentByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	refundID, err := uc.provider.RefundPayment(ctx, payment.XenditInvoiceID, payment.Amount)
	if err != nil {
		return fmt.Errorf("gagal memproses refund ke Xendit: %w", err)
	}

	payment.Status = domain.PaymentStatusRefunded
	payment.XenditRefundID = refundID
	if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
		return err
	}

	// UpdateOrderStatus juga set ticket_units → REFUNDED dalam satu tx
	return uc.repo.UpdateOrderStatus(ctx, orderID, domain.OrderStatusRefunded)
}
