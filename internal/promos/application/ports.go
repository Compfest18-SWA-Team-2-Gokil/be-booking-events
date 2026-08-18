package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/domain"
)

type PromoRepository interface {
	CreatePromo(ctx context.Context, p *domain.Promo) (*domain.Promo, error)
	UpdatePromo(ctx context.Context, p *domain.Promo) (*domain.Promo, error)
	ListPromos(ctx context.Context, onlyActive bool) ([]*domain.Promo, error)
	GetPromoByCode(ctx context.Context, code string) (*domain.Promo, error)
	GetPromoByID(ctx context.Context, id string) (*domain.Promo, error)
	IncrementUsage(ctx context.Context, id string) error
	TogglePromoActive(ctx context.Context, id string, active bool) error
	DeletePromo(ctx context.Context, id string) error
}
