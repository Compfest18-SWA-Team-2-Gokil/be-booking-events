package application

import (
	"context"
	"strings"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/domain"
)

type AdminPromosUseCase struct {
	repo PromoRepository
}

func NewAdminPromosUseCase(repo PromoRepository) *AdminPromosUseCase {
	return &AdminPromosUseCase{repo: repo}
}

type CreatePromoInput struct {
	Code              string              `json:"code"`
	Title             string              `json:"title"`
	Description       string              `json:"description"`
	Type              domain.PromoType    `json:"type"` // "VOUCHER" atau "PROMO"
	EventID           *string             `json:"event_id,omitempty"`
	DiscountType      domain.DiscountType `json:"discount_type"`
	DiscountValue     int64               `json:"discount_value"`
	MinOrderAmount    int64               `json:"min_order_amount"`
	MaxDiscountAmount int64               `json:"max_discount_amount"`
	MaxUsage          int                 `json:"max_usage"`
	IsActive          bool                `json:"is_active"`
	StartDate         *time.Time          `json:"start_date,omitempty"`
	EndDate           *time.Time          `json:"end_date,omitempty"`
}

type UpdatePromoInput struct {
	Code              string              `json:"code"`
	Title             string              `json:"title"`
	Description       string              `json:"description"`
	Type              domain.PromoType    `json:"type"`
	EventID           *string             `json:"event_id,omitempty"`
	DiscountType      domain.DiscountType `json:"discount_type"`
	DiscountValue     int64               `json:"discount_value"`
	MinOrderAmount    int64               `json:"min_order_amount"`
	MaxDiscountAmount int64               `json:"max_discount_amount"`
	MaxUsage          int                 `json:"max_usage"`
	IsActive          bool                `json:"is_active"`
	StartDate         *time.Time          `json:"start_date,omitempty"`
	EndDate           *time.Time          `json:"end_date,omitempty"`
}

func (uc *AdminPromosUseCase) CreatePromo(ctx context.Context, input CreatePromoInput) (*domain.Promo, error) {
	promoType := input.Type
	if promoType == "" {
		if input.EventID != nil && strings.TrimSpace(*input.EventID) != "" {
			promoType = domain.PromoTypePromo
		} else {
			promoType = domain.PromoTypeVoucher
		}
	}

	var eventID *string
	if promoType == domain.PromoTypePromo && input.EventID != nil && strings.TrimSpace(*input.EventID) != "" {
		cleaned := strings.TrimSpace(*input.EventID)
		eventID = &cleaned
	}

	p := &domain.Promo{
		Code:              strings.ToUpper(strings.TrimSpace(input.Code)),
		Title:             strings.TrimSpace(input.Title),
		Description:       strings.TrimSpace(input.Description),
		Type:              promoType,
		EventID:           eventID,
		DiscountType:      input.DiscountType,
		DiscountValue:     input.DiscountValue,
		MinOrderAmount:    input.MinOrderAmount,
		MaxDiscountAmount: input.MaxDiscountAmount,
		MaxUsage:          input.MaxUsage,
		IsActive:          true,
		StartDate:         input.StartDate,
		EndDate:           input.EndDate,
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return uc.repo.CreatePromo(ctx, p)
}

func (uc *AdminPromosUseCase) UpdatePromo(ctx context.Context, id string, input UpdatePromoInput) (*domain.Promo, error) {
	promoType := input.Type
	if promoType == "" {
		if input.EventID != nil && strings.TrimSpace(*input.EventID) != "" {
			promoType = domain.PromoTypePromo
		} else {
			promoType = domain.PromoTypeVoucher
		}
	}

	var eventID *string
	if promoType == domain.PromoTypePromo && input.EventID != nil && strings.TrimSpace(*input.EventID) != "" {
		cleaned := strings.TrimSpace(*input.EventID)
		eventID = &cleaned
	}

	p := &domain.Promo{
		ID:                id,
		Code:              strings.ToUpper(strings.TrimSpace(input.Code)),
		Title:             strings.TrimSpace(input.Title),
		Description:       strings.TrimSpace(input.Description),
		Type:              promoType,
		EventID:           eventID,
		DiscountType:      input.DiscountType,
		DiscountValue:     input.DiscountValue,
		MinOrderAmount:    input.MinOrderAmount,
		MaxDiscountAmount: input.MaxDiscountAmount,
		MaxUsage:          input.MaxUsage,
		IsActive:          input.IsActive,
		StartDate:         input.StartDate,
		EndDate:           input.EndDate,
	}

	if err := p.Validate(); err != nil {
		return nil, err
	}

	return uc.repo.UpdatePromo(ctx, p)
}

func (uc *AdminPromosUseCase) ListAllPromos(ctx context.Context) ([]*domain.Promo, error) {
	return uc.repo.ListPromos(ctx, false)
}

func (uc *AdminPromosUseCase) ListActivePromos(ctx context.Context) ([]*domain.Promo, error) {
	return uc.repo.ListPromos(ctx, true)
}

func (uc *AdminPromosUseCase) ToggleActive(ctx context.Context, id string, active bool) error {
	return uc.repo.TogglePromoActive(ctx, id, active)
}

func (uc *AdminPromosUseCase) DeletePromo(ctx context.Context, id string) error {
	return uc.repo.DeletePromo(ctx, id)
}
