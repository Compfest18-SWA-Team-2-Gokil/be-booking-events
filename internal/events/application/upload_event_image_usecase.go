package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/events/domain"
)

var (
	ErrFileTooLarge    = errors.New("ukuran file maksimal 5MB")
	ErrInvalidFileType = errors.New("tipe file tidak didukung, gunakan JPEG, PNG, atau WebP")
)

const maxFileSize = 5 * 1024 * 1024 // 5MB

var allowedMIMETypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

type UploadEventImageUseCase struct {
	repo    EventRepository
	storage StorageProvider
}

func NewUploadEventImageUseCase(repo EventRepository, storage StorageProvider) *UploadEventImageUseCase {
	return &UploadEventImageUseCase{repo: repo, storage: storage}
}

type UploadEventImageInput struct {
	EventID     string
	OrganizerID string
	Data        []byte
	ContentType string
}

type UploadEventImageOutput struct {
	ImageURL string `json:"image_url"`
}

func (uc *UploadEventImageUseCase) Execute(ctx context.Context, input UploadEventImageInput) (*UploadEventImageOutput, error) {
	if len(input.Data) > maxFileSize {
		return nil, ErrFileTooLarge
	}

	ext, ok := allowedMIMETypes[input.ContentType]
	if !ok {
		return nil, ErrInvalidFileType
	}

	// Pastikan event ada dan milik organizer yang benar.
	event, err := uc.repo.GetEvent(ctx, input.EventID)
	if err != nil {
		return nil, err
	}
	if event.OrganizerID != input.OrganizerID {
		return nil, domain.ErrEventNotFound
	}

	objectKey := fmt.Sprintf("events/%s/%d.%s", input.EventID, time.Now().UnixMilli(), ext)

	publicURL, err := uc.storage.UploadImage(ctx, objectKey, input.Data, input.ContentType)
	if err != nil {
		return nil, fmt.Errorf("gagal upload gambar: %w", err)
	}

	if err := uc.repo.UpdateEventImageURL(ctx, input.EventID, publicURL); err != nil {
		return nil, err
	}

	return &UploadEventImageOutput{ImageURL: publicURL}, nil
}
