package application

import (
	"context"
	"strings"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/domain"
)

type ValidatePromoUseCase struct {
	repo PromoRepository
}

func NewValidatePromoUseCase(repo PromoRepository) *ValidatePromoUseCase {
	return &ValidatePromoUseCase{repo: repo}
}

type ValidatePromoInput struct {
	Code        string `json:"code"`
	TotalAmount int64  `json:"total_amount"`
	EventID     string `json:"event_id,omitempty"`
}

type ValidatePromoOutput struct {
	Code           string              `json:"code"`
	Title          string              `json:"title"`
	EventID        *string             `json:"event_id,omitempty"`
	EventName      string              `json:"event_name,omitempty"`
	DiscountType   domain.DiscountType `json:"discount_type"`
	DiscountValue  int64               `json:"discount_value"`
	DiscountAmount int64               `json:"discount_amount"`
	FinalAmount    int64               `json:"final_amount"`
}

func (uc *ValidatePromoUseCase) Execute(ctx context.Context, input ValidatePromoInput) (*ValidatePromoOutput, error) {
	code := strings.ToUpper(strings.TrimSpace(input.Code))
	if code == "" {
		return nil, domain.ErrInvalidPromoCode
	}

	promo, err := uc.repo.GetPromoByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	discount, err := promo.CalculateDiscountForEvent(input.TotalAmount, input.EventID)
	if err != nil {
		return nil, err
	}

	finalAmount := input.TotalAmount - discount
	if finalAmount < 0 {
		finalAmount = 0
	}

	return &ValidatePromoOutput{
		Code:           promo.Code,
		Title:          promo.Title,
		EventID:        promo.EventID,
		EventName:      promo.EventName,
		DiscountType:   promo.DiscountType,
		DiscountValue:  promo.DiscountValue,
		DiscountAmount: discount,
		FinalAmount:    finalAmount,
	}, nil
}
