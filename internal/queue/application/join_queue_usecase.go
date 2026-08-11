package application

import (
	"context"
	"fmt"
)

type JoinQueueUseCase struct {
	repo QueueRepository
}

func NewJoinQueueUseCase(repo QueueRepository) *JoinQueueUseCase {
	return &JoinQueueUseCase{repo: repo}
}

type JoinQueueInput struct {
	EventID string
	UserID  string
}

type JoinQueueOutput struct {
	Position int64 `json:"position"`
}

func (uc *JoinQueueUseCase) Execute(ctx context.Context, input JoinQueueInput) (*JoinQueueOutput, error) {
	if input.EventID == "" || input.UserID == "" {
		return nil, fmt.Errorf("eventID dan userID harus diisi")
	}

	if err := uc.repo.AddActiveQueue(ctx, input.EventID); err != nil {
		return nil, err
	}

	position, err := uc.repo.Join(ctx, input.EventID, input.UserID)
	if err != nil {
		return nil, err
	}

	return &JoinQueueOutput{Position: position}, nil
}
