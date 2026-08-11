package application

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type EventRepository interface {
	CreateEvent(ctx context.Context, event *domain.Event) error
	ListEvents(ctx context.Context) ([]*domain.Event, error)
	GetEvent(ctx context.Context, eventID string) (*domain.Event, error)

	CreateTicketType(ctx context.Context, tt *domain.TicketType) error
	ListTicketTypes(ctx context.Context, eventID string) ([]*domain.TicketType, error)
	GetTicketType(ctx context.Context, ticketTypeID string) (*domain.TicketType, error)

	// ProvisionUnits membuat N ticket_unit rows untuk satu TicketType.
	// Untuk GA: seat_label = NULL. Untuk SEATED: butuh seat labels (fase berikutnya).
	ProvisionUnits(ctx context.Context, ticketTypeID string, quantity int) error

	// CountProvisionedUnits menghitung berapa unit sudah dibuat untuk TicketType ini.
	CountProvisionedUnits(ctx context.Context, ticketTypeID string) (int, error)
}
