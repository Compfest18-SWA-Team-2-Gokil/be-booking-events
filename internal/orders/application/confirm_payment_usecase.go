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

	// === PERUBAHAN BARU: Validasi Idempotensi & Pengaman Urutan Webhook ===
	// Pengecekan dilakukan menggunakan helper method di bawah untuk menghindari override status final.
	if uc.shouldSkipStateTransition(order.Status, payment.Status, input.Status) {
		return nil
	}
	// === AKHIR PERUBAHAN BARU ===

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

// === PERUBAHAN BARU: Helper Method Di Tempat Paling Bawah ===
// shouldSkipStateTransition memvalidasi apakah status transisi dari webhook aman diproses.
// Ini mencegah status EXPIRED dari webhook membatalkan transaksi yang sudah sukses (PAID/REFUNDED).
func (uc *ConfirmPaymentUseCase) shouldSkipStateTransition(orderStatus domain.OrderStatus, paymentStatus domain.PaymentStatus, webhookStatus string) bool {
	// Jika order sudah PAID, REFUND_REQUESTED, atau REFUNDED:
	// - Baik webhook PAID (redundant) maupun EXPIRED (out-of-order) harus diabaikan demi konsistensi data.
	if orderStatus == domain.OrderStatusPaid || 
	   orderStatus == domain.OrderStatusRefundRequested || 
	   orderStatus == domain.OrderStatusRefunded {
		return true
	}

	// Jika order sudah CANCELLED:
	// - Abaikan semua event webhook lanjutan untuk order ini.
	if orderStatus == domain.OrderStatusCancelled {
		return true
	}

	return false
}
// === AKHIR PERUBAHAN BARU ===
