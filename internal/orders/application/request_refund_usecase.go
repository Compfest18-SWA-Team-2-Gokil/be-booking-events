package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/audit"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
)

type RequestRefundUseCase struct {
	repo        OrderRepository
	auditLogger *audit.Logger
}

func NewRequestRefundUseCase(repo OrderRepository, auditLogger *audit.Logger) *RequestRefundUseCase {
	return &RequestRefundUseCase{repo: repo, auditLogger: auditLogger}
}

func (uc *RequestRefundUseCase) Execute(ctx context.Context, orderID, buyerID string) error {
	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}

	if order.BuyerID != buyerID {
		return domain.ErrOrderNotFound
	}

	if order.Status != domain.OrderStatusPaid {
		return domain.ErrOrderNotPaid
	}

	// Validasi H-1: Pengajuan refund maksimal H-1 (24 jam) sebelum event dimulai.
	eventDate, err := uc.repo.GetEventDate(ctx, order.EventID)
	if err == nil && !eventDate.IsZero() {
		refundDeadline := eventDate.Add(-24 * time.Hour)
		if time.Now().After(refundDeadline) {
			return domain.ErrRefundDeadlinePassed
		}
	}

	// PRD-09 Status Blocker: tolak refund jika ada tiket yang sudah di-scan gerbang.
	admitted, err := uc.repo.HasAdmittedUnits(ctx, orderID)
	if err != nil {
		return err
	}
	if admitted {
		return domain.ErrTicketAlreadyAdmitted
	}

	if err := uc.repo.UpdateOrderStatus(ctx, order.ID, domain.OrderStatusRefundRequested); err != nil {
		return err
	}

	// Audit: refund request
	if uc.auditLogger != nil {
		uc.auditLogger.Log(ctx, audit.Entry{
			ActorID:    buyerID,
			ActorRole:  "BUYER",
			EntityType: "order",
			EntityID:   order.ID,
			Action:     "REFUND_REQUESTED",
			FromStatus: string(domain.OrderStatusPaid),
			ToStatus:   string(domain.OrderStatusRefundRequested),
		})
	}

	return nil
}
