package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/audit/domain"
)

type AuditRepository interface {
	Insert(ctx context.Context, log *domain.AuditLog) error
	ListByEntity(ctx context.Context, entityName, entityID string) ([]*domain.AuditLog, error)
	ListRecent(ctx context.Context, limit int) ([]*domain.AuditLog, error)
}
