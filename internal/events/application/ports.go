package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, event *domain.Event) error
	GetEvent(ctx context.Context, eventID string) (*domain.Event, error)
	ListEvents(ctx context.Context, filter ListEventsFilter) ([]*domain.Event, error)
	UpdateEvent(ctx context.Context, event *domain.Event) error
	DeleteEvent(ctx context.Context, eventID string) error

	CreateTicketType(ctx context.Context, tt *domain.TicketType) error
	ListTicketTypes(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	GetTicketType(ctx context.Context, ticketTypeID string) (*domain.TicketType, error)
	UpdateTicketType(ctx context.Context, tt *domain.TicketType) error
	DeleteTicketType(ctx context.Context, ticketTypeID string) error

	ProvisionUnits(ctx context.Context, ticketTypeID string, quantity int) error
	CountProvisionedUnits(ctx context.Context, ticketTypeID string) (int, error)

	// CountSoldUnits menghitung unit dengan status selain AVAILABLE untuk satu TicketType.
	// Digunakan untuk enforce data lock PRD-02.
	CountSoldUnits(ctx context.Context, ticketTypeID string) (int, error)

	// HasNonAvailableUnits cek apakah event punya unit yang tidak AVAILABLE.
	// Digunakan sebelum delete event.
	HasNonAvailableUnits(ctx context.Context, eventID string) (bool, error)

	UpdateEventImageURL(ctx context.Context, eventID, imageURL string) error
}

// StorageProvider abstraksi ke object storage (MinIO, S3, dll).
type StorageProvider interface {
	UploadImage(ctx context.Context, objectKey string, data []byte, contentType string) (publicURL string, err error)
}

// ListEventsFilter parameter opsional untuk filter list events.
type ListEventsFilter struct {
	Category string // kosong = semua kategori
	Page     int    // 1-based, 0 = default ke 1
	Limit    int    // 0 = default ke 20
}
