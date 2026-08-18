package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/domain"
)

type ValidateQueueTokenUseCase struct {
	signer TokenSigner
}

func NewValidateQueueTokenUseCase(signer TokenSigner) *ValidateQueueTokenUseCase {
	return &ValidateQueueTokenUseCase{signer: signer}
}

// Execute memverifikasi queue token dan mengembalikan payload-nya.
// Dipanggil sebagai middleware sebelum checkout endpoint.
func (uc *ValidateQueueTokenUseCase) Execute(_ context.Context, tokenStr string) (*domain.QueueToken, error) {
	token, err := uc.signer.Verify(tokenStr)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if token.IsExpired() {
		return nil, domain.ErrTokenExpired
	}

	return token, nil
}
