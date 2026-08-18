package infrastructure

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPromoRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPromoRepository(pool *pgxpool.Pool) *PostgresPromoRepository {
	return &PostgresPromoRepository{pool: pool}
}

var _ application.PromoRepository = (*PostgresPromoRepository)(nil)

func (r *PostgresPromoRepository) CreatePromo(ctx context.Context, p *domain.Promo) (*domain.Promo, error) {
	out := &domain.Promo{}
	var discType, promoType string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO promos (
			code, title, description, type, event_id, discount_type, discount_value,
			min_order_amount, max_discount_amount, max_usage, used_count, is_active,
			start_date, end_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, $11, $12, $13)
		RETURNING id, code, title, description, type, event_id, discount_type, discount_value,
		          min_order_amount, max_discount_amount, max_usage, used_count, is_active,
		          start_date, end_date, created_at, updated_at
	`, p.Code, p.Title, p.Description, string(p.Type), p.EventID, string(p.DiscountType), p.DiscountValue,
		p.MinOrderAmount, p.MaxDiscountAmount, p.MaxUsage, p.IsActive,
		p.StartDate, p.EndDate,
	).Scan(
		&out.ID, &out.Code, &out.Title, &out.Description, &promoType, &out.EventID, &discType, &out.DiscountValue,
		&out.MinOrderAmount, &out.MaxDiscountAmount, &out.MaxUsage, &out.UsedCount, &out.IsActive,
		&out.StartDate, &out.EndDate, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create promo: %w", err)
	}
	out.DiscountType = domain.DiscountType(discType)
	out.Type = domain.PromoType(promoType)

	if out.EventID != nil && *out.EventID != "" {
		_ = r.pool.QueryRow(ctx, `SELECT name FROM events WHERE id = $1`, *out.EventID).Scan(&out.EventName)
	}

	return out, nil
}

func (r *PostgresPromoRepository) UpdatePromo(ctx context.Context, p *domain.Promo) (*domain.Promo, error) {
	out := &domain.Promo{}
	var discType, promoType string
	err := r.pool.QueryRow(ctx, `
		UPDATE promos
		SET code = $2,
		    title = $3,
		    description = $4,
		    type = $5,
		    event_id = $6,
		    discount_type = $7,
		    discount_value = $8,
		    min_order_amount = $9,
		    max_discount_amount = $10,
		    max_usage = $11,
		    is_active = $12,
		    start_date = $13,
		    end_date = $14,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, code, title, description, type, event_id, discount_type, discount_value,
		          min_order_amount, max_discount_amount, max_usage, used_count, is_active,
		          start_date, end_date, created_at, updated_at
	`, p.ID, p.Code, p.Title, p.Description, string(p.Type), p.EventID, string(p.DiscountType), p.DiscountValue,
		p.MinOrderAmount, p.MaxDiscountAmount, p.MaxUsage, p.IsActive,
		p.StartDate, p.EndDate,
	).Scan(
		&out.ID, &out.Code, &out.Title, &out.Description, &promoType, &out.EventID, &discType, &out.DiscountValue,
		&out.MinOrderAmount, &out.MaxDiscountAmount, &out.MaxUsage, &out.UsedCount, &out.IsActive,
		&out.StartDate, &out.EndDate, &out.CreatedAt, &out.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrPromoNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update promo: %w", err)
	}
	out.DiscountType = domain.DiscountType(discType)
	out.Type = domain.PromoType(promoType)

	if out.EventID != nil && *out.EventID != "" {
		_ = r.pool.QueryRow(ctx, `SELECT name FROM events WHERE id = $1`, *out.EventID).Scan(&out.EventName)
	}

	return out, nil
}

func (r *PostgresPromoRepository) ListPromos(ctx context.Context, onlyActive bool) ([]*domain.Promo, error) {
	query := `
		SELECT p.id, p.code, p.title, p.description, p.type, p.event_id, COALESCE(e.name, '') as event_name,
		       p.discount_type, p.discount_value, p.min_order_amount, p.max_discount_amount,
		       p.max_usage, p.used_count, p.is_active, p.start_date, p.end_date, p.created_at, p.updated_at
		FROM promos p
		LEFT JOIN events e ON e.id = p.event_id
	`
	if onlyActive {
		query += " WHERE p.is_active = TRUE"
	}
	query += " ORDER BY p.created_at DESC"

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list promos: %w", err)
	}
	defer rows.Close()

	promos := make([]*domain.Promo, 0)
	for rows.Next() {
		p := &domain.Promo{}
		var discType, promoType string
		if err := rows.Scan(
			&p.ID, &p.Code, &p.Title, &p.Description, &promoType, &p.EventID, &p.EventName,
			&discType, &p.DiscountValue, &p.MinOrderAmount, &p.MaxDiscountAmount,
			&p.MaxUsage, &p.UsedCount, &p.IsActive, &p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan promo: %w", err)
		}
		p.DiscountType = domain.DiscountType(discType)
		p.Type = domain.PromoType(promoType)
		promos = append(promos, p)
	}
	return promos, rows.Err()
}

func (r *PostgresPromoRepository) GetPromoByCode(ctx context.Context, code string) (*domain.Promo, error) {
	p := &domain.Promo{}
	var discType, promoType string
	err := r.pool.QueryRow(ctx, `
		SELECT p.id, p.code, p.title, p.description, p.type, p.event_id, COALESCE(e.name, '') as event_name,
		       p.discount_type, p.discount_value, p.min_order_amount, p.max_discount_amount,
		       p.max_usage, p.used_count, p.is_active, p.start_date, p.end_date, p.created_at, p.updated_at
		FROM promos p
		LEFT JOIN events e ON e.id = p.event_id
		WHERE UPPER(p.code) = UPPER($1)
	`, code).Scan(
		&p.ID, &p.Code, &p.Title, &p.Description, &promoType, &p.EventID, &p.EventName,
		&discType, &p.DiscountValue, &p.MinOrderAmount, &p.MaxDiscountAmount,
		&p.MaxUsage, &p.UsedCount, &p.IsActive, &p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrPromoNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get promo by code: %w", err)
	}
	p.DiscountType = domain.DiscountType(discType)
	p.Type = domain.PromoType(promoType)
	return p, nil
}

func (r *PostgresPromoRepository) GetPromoByID(ctx context.Context, id string) (*domain.Promo, error) {
	p := &domain.Promo{}
	var discType, promoType string
	err := r.pool.QueryRow(ctx, `
		SELECT p.id, p.code, p.title, p.description, p.type, p.event_id, COALESCE(e.name, '') as event_name,
		       p.discount_type, p.discount_value, p.min_order_amount, p.max_discount_amount,
		       p.max_usage, p.used_count, p.is_active, p.start_date, p.end_date, p.created_at, p.updated_at
		FROM promos p
		LEFT JOIN events e ON e.id = p.event_id
		WHERE p.id = $1
	`, id).Scan(
		&p.ID, &p.Code, &p.Title, &p.Description, &promoType, &p.EventID, &p.EventName,
		&discType, &p.DiscountValue, &p.MinOrderAmount, &p.MaxDiscountAmount,
		&p.MaxUsage, &p.UsedCount, &p.IsActive, &p.StartDate, &p.EndDate, &p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, domain.ErrPromoNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get promo by id: %w", err)
	}
	p.DiscountType = domain.DiscountType(discType)
	p.Type = domain.PromoType(promoType)
	return p, nil
}

func (r *PostgresPromoRepository) IncrementUsage(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE promos
		SET used_count = used_count + 1, updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *PostgresPromoRepository) TogglePromoActive(ctx context.Context, id string, active bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE promos
		SET is_active = $2, updated_at = NOW()
		WHERE id = $1
	`, id, active)
	return err
}

func (r *PostgresPromoRepository) DeletePromo(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM promos WHERE id = $1`, id)
	return err
}
