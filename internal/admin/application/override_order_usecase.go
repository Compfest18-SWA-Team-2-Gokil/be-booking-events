package application

import (
	"context"
	"errors"
)

var ErrInvalidOverrideStatus = errors.New("status override tidak valid: gunakan PAID, CANCELLED, atau REFUNDED")

var allowedOverrideStatuses = map[string]bool{
	"PAID":      true,
	"CANCELLED": true,
	"REFUNDED":  true,
}

type OverrideOrderUseCase struct {
	repo AdminRepository
}

func NewOverrideOrderUseCase(repo AdminRepository) *OverrideOrderUseCase {
	return &OverrideOrderUseCase{repo: repo}
}

func (uc *OverrideOrderUseCase) Execute(ctx context.Context, orderID, adminID, newStatus, reason string) error {
	if !allowedOverrideStatuses[newStatus] {
		return ErrInvalidOverrideStatus
	}
	return uc.repo.OverrideOrderStatus(ctx, orderID, adminID, newStatus, reason)
}
