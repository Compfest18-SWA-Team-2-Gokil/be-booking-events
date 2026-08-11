package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/queue/application"
	"github.com/ebk-tech/be-booking-events/internal/queue/domain"
)

func TestValidateQueueTokenUseCase_Execute_Valid(t *testing.T) {
	token := &domain.QueueToken{
		UserID:    "user-1",
		EventID:   "event-1",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	signer := &fakeTokenSigner{token: token}
	uc := application.NewValidateQueueTokenUseCase(signer)

	result, err := uc.Execute(context.Background(), "any-token-string")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserID != "user-1" {
		t.Errorf("user_id = %s, want user-1", result.UserID)
	}
}

func TestValidateQueueTokenUseCase_Execute_InvalidSignature(t *testing.T) {
	signer := &fakeTokenSigner{verifyErr: domain.ErrInvalidToken}
	uc := application.NewValidateQueueTokenUseCase(signer)

	_, err := uc.Execute(context.Background(), "bad-token")
	if err != domain.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestValidateQueueTokenUseCase_Execute_Expired(t *testing.T) {
	token := &domain.QueueToken{
		UserID:    "user-1",
		EventID:   "event-1",
		IssuedAt:  time.Now().Add(-20 * time.Minute),
		ExpiresAt: time.Now().Add(-10 * time.Minute), // sudah expired
	}
	signer := &fakeTokenSigner{token: token}
	uc := application.NewValidateQueueTokenUseCase(signer)

	_, err := uc.Execute(context.Background(), "expired-token")
	if err != domain.ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
}

func TestJoinQueueUseCase_Execute_Success(t *testing.T) {
	repo := newFakeQueueRepo()
	uc := application.NewJoinQueueUseCase(repo)

	out, err := uc.Execute(context.Background(), application.JoinQueueInput{
		EventID: "event-1",
		UserID:  "user-1",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Position != 0 {
		t.Errorf("position = %d, want 0 (user pertama)", out.Position)
	}
	if !repo.activeQueues["event-1"] {
		t.Error("event-1 seharusnya masuk active queues")
	}
}

func TestJoinQueueUseCase_Execute_SecondUser(t *testing.T) {
	repo := newFakeQueueRepo()
	uc := application.NewJoinQueueUseCase(repo)

	uc.Execute(context.Background(), application.JoinQueueInput{EventID: "event-1", UserID: "user-1"})
	out, err := uc.Execute(context.Background(), application.JoinQueueInput{EventID: "event-1", UserID: "user-2"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Position != 1 {
		t.Errorf("position = %d, want 1", out.Position)
	}
}

func TestJoinQueueUseCase_Execute_EmptyInput(t *testing.T) {
	uc := application.NewJoinQueueUseCase(newFakeQueueRepo())

	_, err := uc.Execute(context.Background(), application.JoinQueueInput{})
	if err == nil {
		t.Fatal("expected error untuk input kosong")
	}
}
