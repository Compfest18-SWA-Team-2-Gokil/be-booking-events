package application

import "context"

type ListDisputesUseCase struct {
	repo AdminRepository
}

func NewListDisputesUseCase(repo AdminRepository) *ListDisputesUseCase {
	return &ListDisputesUseCase{repo: repo}
}

func (uc *ListDisputesUseCase) Execute(ctx context.Context) ([]DisputeOrder, error) {
	return uc.repo.ListDisputes(ctx)
}
