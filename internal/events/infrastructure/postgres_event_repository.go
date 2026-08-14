package infrastructure

import (
	"context"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/events/application"
	"github.com/ebk-tech/be-booking-events/internal/events/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresEventRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEventRepository(pool *pgxpool.Pool) *PostgresEventRepository {
	return &PostgresEventRepository{pool: pool}
}

var _ application.EventRepository = (*PostgresEventRepository)(nil)

func (r *PostgresEventRepository) CreateEvent(ctx context.Context, e *domain.Event) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO events (name, date, location, organizer_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, e.Name, e.Date, e.Location, nullableStr(e.OrganizerID)).Scan(&e.ID)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) UpdateEventImageURL(ctx context.Context, eventID, imageURL string) error {
	_, err := r.pool.Exec(ctx, `UPDATE events SET image_url = $1 WHERE id = $2`, imageURL, eventID)
	return err
}

func (r *PostgresEventRepository) ListEvents(ctx context.Context) ([]*domain.Event, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, COALESCE(organizer_id::text, ''), name, date, location, COALESCE(image_url, '')
		FROM events
		ORDER BY date ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e := &domain.Event{}
		if err := rows.Scan(&e.ID, &e.OrganizerID, &e.Name, &e.Date, &e.Location, &e.ImageURL); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (r *PostgresEventRepository) GetEvent(ctx context.Context, eventID string) (*domain.Event, error) {
	e := &domain.Event{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(organizer_id::text, ''), name, date, location, COALESCE(image_url, '')
		FROM events WHERE id = $1
	`, eventID).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.Date, &e.Location, &e.ImageURL)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	return e, nil
}

func (r *PostgresEventRepository) CreateTicketType(ctx context.Context, tt *domain.TicketType) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ticket_types (event_id, name, price, kind, total_quota)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, tt.EventID, tt.Name, tt.Price, string(tt.Kind), tt.TotalQuota).Scan(&tt.ID)
	if err != nil {
		return fmt.Errorf("create ticket type: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) ListTicketTypes(ctx context.Context, eventID string) ([]*domain.TicketType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, event_id, name, price, kind, total_quota
		FROM ticket_types
		WHERE event_id = $1
		ORDER BY created_at ASC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list ticket types: %w", err)
	}
	defer rows.Close()

	var types []*domain.TicketType
	for rows.Next() {
		tt := &domain.TicketType{}
		var kind string
		if err := rows.Scan(&tt.ID, &tt.EventID, &tt.Name, &tt.Price, &kind, &tt.TotalQuota); err != nil {
			return nil, fmt.Errorf("scan ticket type: %w", err)
		}
		tt.Kind = domain.Kind(kind)
		types = append(types, tt)
	}
	return types, rows.Err()
}

func (r *PostgresEventRepository) GetTicketType(ctx context.Context, ticketTypeID string) (*domain.TicketType, error) {
	tt := &domain.TicketType{}
	var kind string
	err := r.pool.QueryRow(ctx, `
		SELECT id, event_id, name, price, kind, total_quota
		FROM ticket_types WHERE id = $1
	`, ticketTypeID).Scan(&tt.ID, &tt.EventID, &tt.Name, &tt.Price, &kind, &tt.TotalQuota)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrTicketTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get ticket type: %w", err)
	}
	tt.Kind = domain.Kind(kind)
	return tt, nil
}

// ProvisionUnits bulk insert ticket_unit rows menggunakan unnest untuk efisiensi.
func (r *PostgresEventRepository) ProvisionUnits(ctx context.Context, ticketTypeID string, quantity int) error {
	ids := make([]string, quantity)
	for i := range ids {
		ids[i] = ticketTypeID
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO ticket_units (ticket_type_id)
		SELECT unnest($1::uuid[])
	`, ids)
	if err != nil {
		return fmt.Errorf("provision units: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) CountProvisionedUnits(ctx context.Context, ticketTypeID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ticket_units WHERE ticket_type_id = $1
	`, ticketTypeID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count provisioned units: %w", err)
	}
	return count, nil
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
