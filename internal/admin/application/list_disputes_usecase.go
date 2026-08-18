package application

import "context"

type ListDisputesUseCase struct {
	repo AdminRepository
}

func NewListDisputesUseCase(repo AdminRepository) *ListDisputesUseCase {
	return &ListDisputesUseCase{repo: repo}
}

func (uc *ListDisputesUseCase) Execute(ctx context.Context, limit, offset int) ([]DisputeOrder, int, error) {
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}
	return uc.repo.ListDisputes(ctx, limit, offset)
}
