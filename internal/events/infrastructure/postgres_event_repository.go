package infrastructure

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/events/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/events/domain"
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
		INSERT INTO events (name, description, category, date, location, organizer_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, e.Name, e.Description, string(e.Category), e.Date, e.Location, nullableStr(e.OrganizerID)).Scan(&e.ID)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) GetEvent(ctx context.Context, eventID string) (*domain.Event, error) {
	e := &domain.Event{}
	var category string
	err := r.pool.QueryRow(ctx, `
		SELECT id, COALESCE(organizer_id::text, ''), name, COALESCE(description, ''), COALESCE(category, 'music'),
		       date, location, COALESCE(image_url, '')
		FROM events WHERE id = $1
	`, eventID).Scan(&e.ID, &e.OrganizerID, &e.Name, &e.Description, &category, &e.Date, &e.Location, &e.ImageURL)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	e.Category = domain.Category(category)
	return e, nil
}


func (r *PostgresEventRepository) ListEvents(ctx context.Context, filter application.ListEventsFilter) ([]*domain.Event, int, error) {
	offset := (filter.Page - 1) * filter.Limit
	var total int

	if filter.Category != "" {
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM events WHERE category = $1`, filter.Category).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("count events: %w", err)
		}
	} else {
		err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM events`).Scan(&total)
		if err != nil {
			return nil, 0, fmt.Errorf("count events: %w", err)
		}
	}

	var rows pgx.Rows
	var err error
	if filter.Category != "" {
		rows, err = r.pool.Query(ctx, `
			SELECT id, COALESCE(organizer_id::text, ''), name, COALESCE(description, ''), COALESCE(category, 'music'),
			       date, location, COALESCE(image_url, '')
			FROM events
			WHERE category = $1
			ORDER BY date ASC
			LIMIT $2 OFFSET $3
		`, filter.Category, filter.Limit, offset)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, COALESCE(organizer_id::text, ''), name, COALESCE(description, ''), COALESCE(category, 'music'),
			       date, location, COALESCE(image_url, '')
			FROM events
			ORDER BY date ASC
			LIMIT $1 OFFSET $2
		`, filter.Limit, offset)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	var events []*domain.Event
	for rows.Next() {
		e := &domain.Event{}
		var category string
		if err := rows.Scan(&e.ID, &e.OrganizerID, &e.Name, &e.Description, &category, &e.Date, &e.Location, &e.ImageURL); err != nil {
			return nil, 0, fmt.Errorf("scan event: %w", err)
		}
		e.Category = domain.Category(category)
		events = append(events, e)
	}
	return events, total, rows.Err()
}

func (r *PostgresEventRepository) UpdateEvent(ctx context.Context, e *domain.Event) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE events
		SET name = $1, description = $2, category = $3, date = $4, location = $5, updated_at = NOW()
		WHERE id = $6
	`, e.Name, e.Description, string(e.Category), e.Date, e.Location, e.ID)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) DeleteEvent(ctx context.Context, eventID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Hapus semua ticket_units untuk event ini dulu.
	_, err = tx.Exec(ctx, `
		DELETE FROM ticket_units
		WHERE ticket_type_id IN (SELECT id FROM ticket_types WHERE event_id = $1)
	`, eventID)
	if err != nil {
		return fmt.Errorf("delete ticket units: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM ticket_types WHERE event_id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("delete ticket types: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM events WHERE id = $1`, eventID)
	if err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *PostgresEventRepository) HasNonAvailableUnits(ctx context.Context, eventID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ticket_units tu
		JOIN ticket_types tt ON tu.ticket_type_id = tt.id
		WHERE tt.event_id = $1 AND tu.status != 'AVAILABLE'
	`, eventID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("has non-available units: %w", err)
	}
	return count > 0, nil
}

func (r *PostgresEventRepository) UpdateEventImageURL(ctx context.Context, eventID, imageURL string) error {
	_, err := r.pool.Exec(ctx, `UPDATE events SET image_url = $1 WHERE id = $2`, imageURL, eventID)
	return err
}

func (r *PostgresEventRepository) CreateTicketType(ctx context.Context, tt *domain.TicketType) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO ticket_types (event_id, name, price, kind, total_quota)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, tt.EventID, tt.Name, tt.Price, string(tt.Kind), tt.TotalQuota).Scan(&tt.ID)
	if err != nil {
		return fmt.Errorf("create ticket type: %w", err)
	}

	// Generate ticket_units sebanyak total_quota agar tiket siap di-hold / dibeli
	if tt.TotalQuota > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO ticket_units (ticket_type_id, status)
			SELECT $1, 'AVAILABLE'
			FROM generate_series(1, $2)
		`, tt.ID, tt.TotalQuota)
		if err != nil {
			return fmt.Errorf("generate ticket units: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit ticket type: %w", err)
	}

	tt.PriceStatus = "OPEN"
	return nil
}

func (r *PostgresEventRepository) ListTicketTypes(ctx context.Context, eventID string) ([]*domain.TicketType, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT 
			tt.id, 
			tt.event_id, 
			tt.name, 
			tt.price, 
			tt.kind, 
			tt.total_quota, 
			tt.price_status,
			COALESCE(
				CASE 
					WHEN COUNT(tu.id) = 0 THEN tt.total_quota
					ELSE SUM(CASE WHEN tu.status = 'AVAILABLE' THEN 1 ELSE 0 END)
				END, 
				tt.total_quota
			) AS available_quota
		FROM ticket_types tt
		LEFT JOIN ticket_units tu ON tu.ticket_type_id = tt.id
		WHERE tt.event_id = $1
		GROUP BY tt.id, tt.event_id, tt.name, tt.price, tt.kind, tt.total_quota, tt.price_status, tt.created_at
		ORDER BY tt.created_at ASC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list ticket types: %w", err)
	}
	defer rows.Close()

	var types []*domain.TicketType
	for rows.Next() {
		tt := &domain.TicketType{}
		var kind string
		if err := rows.Scan(&tt.ID, &tt.EventID, &tt.Name, &tt.Price, &kind, &tt.TotalQuota, &tt.PriceStatus, &tt.AvailableQuota); err != nil {
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
		SELECT 
			tt.id, 
			tt.event_id, 
			tt.name, 
			tt.price, 
			tt.kind, 
			tt.total_quota, 
			tt.price_status,
			COALESCE(
				CASE 
					WHEN COUNT(tu.id) = 0 THEN tt.total_quota
					ELSE SUM(CASE WHEN tu.status = 'AVAILABLE' THEN 1 ELSE 0 END)
				END, 
				tt.total_quota
			) AS available_quota
		FROM ticket_types tt
		LEFT JOIN ticket_units tu ON tu.ticket_type_id = tt.id
		WHERE tt.id = $1
		GROUP BY tt.id, tt.event_id, tt.name, tt.price, tt.kind, tt.total_quota, tt.price_status
	`, ticketTypeID).Scan(&tt.ID, &tt.EventID, &tt.Name, &tt.Price, &kind, &tt.TotalQuota, &tt.PriceStatus, &tt.AvailableQuota)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrTicketTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get ticket type: %w", err)
	}
	tt.Kind = domain.Kind(kind)
	return tt, nil
}

func (r *PostgresEventRepository) UpdateTicketType(ctx context.Context, tt *domain.TicketType) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE ticket_types
		SET name = $1, price = $2, total_quota = $3, updated_at = NOW()
		WHERE id = $4
	`, tt.Name, tt.Price, tt.TotalQuota, tt.ID)
	if err != nil {
		return fmt.Errorf("update ticket type: %w", err)
	}
	return nil
}

func (r *PostgresEventRepository) DeleteTicketType(ctx context.Context, ticketTypeID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM ticket_units WHERE ticket_type_id = $1`, ticketTypeID)
	if err != nil {
		return fmt.Errorf("delete ticket units: %w", err)
	}

	_, err = tx.Exec(ctx, `DELETE FROM ticket_types WHERE id = $1`, ticketTypeID)
	if err != nil {
		return fmt.Errorf("delete ticket type: %w", err)
	}

	return tx.Commit(ctx)
}

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

func (r *PostgresEventRepository) CountSoldUnits(ctx context.Context, ticketTypeID string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ticket_units
		WHERE ticket_type_id = $1 AND status != 'AVAILABLE'
	`, ticketTypeID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count sold units: %w", err)
	}
	return count, nil
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
