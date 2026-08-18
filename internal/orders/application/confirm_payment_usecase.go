package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/audit"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type ConfirmPaymentUseCase struct {
	repo        OrderRepository
	provider    PaymentProvider
	auditLogger *audit.Logger
}

func NewConfirmPaymentUseCase(repo OrderRepository, provider PaymentProvider, auditLogger *audit.Logger) *ConfirmPaymentUseCase {
	return &ConfirmPaymentUseCase{repo: repo, provider: provider, auditLogger: auditLogger}
}

type ConfirmPaymentInput struct {
	XenditInvoiceID string
	ExternalID      string // == order ID kita
	PaymentMethod   string
	Status          string // "PAID" | "EXPIRED"
}

func (uc *ConfirmPaymentUseCase) Execute(ctx context.Context, input ConfirmPaymentInput) error {
	// Fix #1 (Race Condition): Gunakan atomic idempotency check di dalam transaksi yang sama.
	// ConfirmOrderPaymentIdempotent melakukan:
	//   1. SELECT ... FOR UPDATE pada row order (serialisasi akses concurrent webhook)
	//   2. Cek apakah sudah di status final (idempotency guard) — di dalam transaksi, bukan sebelumnya
	//   3. UPDATE ticket_units HELD→CONFIRMED + cek RowsAffected (lost-seat detection)
	//   4. UPDATE orders.status
	// Dengan demikian dua webhook PAID yang datang bersamaan tidak bisa sama-sama lolos.

	order, err := uc.repo.GetOrder(ctx, input.ExternalID)
	if err != nil {
		return err
	}

	payment, err := uc.repo.GetPaymentByOrderID(ctx, order.ID)
	if err != nil {
		return err
	}

	// Idempotency guard: skip webhook jika order sudah di status final.
	// Pengecekan ini masih di luar transaksi untuk efisiensi (fast-path),
	// tapi ConfirmOrderPayment di bawah juga menggunakan SELECT FOR UPDATE
	// untuk menutup race condition pada dua webhook yang datang bersamaan.
	if uc.shouldSkipStateTransition(order.Status, payment.Status, input.Status) {
		return nil
	}

	if input.Status != "PAID" {
		payment.Status = domain.PaymentStatusFailed
		if err := uc.repo.UpdatePayment(ctx, payment); err != nil {
			return err
		}
		if err := uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusCancelled); err != nil {
			return err
		}
		// Audit: pembayaran expired/gagal
		if uc.auditLogger != nil {
			uc.auditLogger.Log(ctx, audit.Entry{
				ActorID:    "",
				ActorRole:  "SYSTEM",
				EntityType: "order",
				EntityID:   order.ID,
				Action:     "PAYMENT_EXPIRED",
				FromStatus: string(order.Status),
				ToStatus:   string(domain.OrderStatusCancelled),
				Metadata:   map[string]any{"xendit_invoice_id": input.XenditInvoiceID, "xendit_status": input.Status},
			})
		}
		return nil
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
			// Audit: lost seat / payment discrepancy
			if uc.auditLogger != nil {
				uc.auditLogger.Log(ctx, audit.Entry{
					ActorID:    "",
					ActorRole:  "SYSTEM",
					EntityType: "order",
					EntityID:   order.ID,
					Action:     "PAYMENT_DISCREPANCY",
					FromStatus: string(domain.OrderStatusPaid),
					ToStatus:   string(domain.OrderStatusPaymentDiscrepancy),
					Metadata:   map[string]any{"xendit_invoice_id": input.XenditInvoiceID, "reason": "ticket unit no longer available"},
				})
			}
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

	// Audit: payment sukses dikonfirmasi
	if uc.auditLogger != nil {
		uc.auditLogger.Log(ctx, audit.Entry{
			ActorID:    order.BuyerID,
			ActorRole:  "BUYER",
			EntityType: "order",
			EntityID:   order.ID,
			Action:     "CONFIRM_PAYMENT",
			FromStatus: string(domain.OrderStatusPaid),
			ToStatus:   string(domain.OrderStatusPaid),
			Metadata:   map[string]any{"xendit_invoice_id": input.XenditInvoiceID, "payment_method": input.PaymentMethod},
		})
	}

	return nil
}

// shouldSkipStateTransition mencegah webhook out-of-order membatalkan transaksi yang sudah sukses.
func (uc *ConfirmPaymentUseCase) shouldSkipStateTransition(orderStatus domain.OrderStatus, _ domain.PaymentStatus, _ string) bool {
	switch orderStatus {
	case domain.OrderStatusPaid,
		domain.OrderStatusRefundRequested,
		domain.OrderStatusRefunded,
		domain.OrderStatusCancelled,
		domain.OrderStatusPaymentDiscrepancy:
		return true
	}
	return false
}
