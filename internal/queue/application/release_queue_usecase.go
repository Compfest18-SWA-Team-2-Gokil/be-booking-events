package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/domain"
)

// TokenTTL adalah masa berlaku queue token setelah user di-release dari antrean.
const TokenTTL = 10 * time.Minute

type ReleaseQueueUseCase struct {
	repo      QueueRepository
	signer    TokenSigner
	batchSize int64
}

func NewReleaseQueueUseCase(repo QueueRepository, signer TokenSigner, batchSize int64) *ReleaseQueueUseCase {
	return &ReleaseQueueUseCase{repo: repo, signer: signer, batchSize: batchSize}
}

// ExecuteAll memproses semua event yang antreannya aktif.
// Dipanggil oleh background worker secara periodik.
func (uc *ReleaseQueueUseCase) ExecuteAll(ctx context.Context) error {
	eventIDs, err := uc.repo.GetActiveQueues(ctx)
	if err != nil {
		return err
	}

	for _, eventID := range eventIDs {
		released, err := uc.releaseForEvent(ctx, eventID)
		if err != nil {
			slog.Error("release queue error", "event_id", eventID, "err", err)
			continue
		}
		if released > 0 {
			slog.Info("queue released", "event_id", eventID, "released", released)
		}

		// Hapus dari active queues jika antrean sudah kosong.
		size, _ := uc.repo.QueueSize(ctx, eventID)
		if size == 0 {
			_ = uc.repo.RemoveActiveQueue(ctx, eventID)
		}
	}
	return nil
}

func (uc *ReleaseQueueUseCase) releaseForEvent(ctx context.Context, eventID string) (int64, error) {
	userIDs, err := uc.repo.PopTop(ctx, eventID, uc.batchSize)
	if err != nil {
		return 0, err
	}

	now := time.Now()
	for _, userID := range userIDs {
		token := domain.QueueToken{
			UserID:    userID,
			EventID:   eventID,
			IssuedAt:  now,
			ExpiresAt: now.Add(TokenTTL),
		}
		tokenStr, err := uc.signer.Sign(token)
		if err != nil {
			slog.Error("sign queue token error", "user_id", userID, "err", err)
			continue
		}
		if err := uc.repo.SetToken(ctx, eventID, userID, tokenStr, TokenTTL); err != nil {
			slog.Error("set token error", "user_id", userID, "err", err)
		}
	}

	return int64(len(userIDs)), nil
}
