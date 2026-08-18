package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
)

type SearchGateOperatorsUseCase struct {
	repo UserRepository
}

func NewSearchGateOperatorsUseCase(repo UserRepository) *SearchGateOperatorsUseCase {
	return &SearchGateOperatorsUseCase{repo: repo}
}

func (uc *SearchGateOperatorsUseCase) Execute(ctx context.Context, query string) ([]domain.User, error) {
	return uc.repo.SearchGateOperators(ctx, query)
}
