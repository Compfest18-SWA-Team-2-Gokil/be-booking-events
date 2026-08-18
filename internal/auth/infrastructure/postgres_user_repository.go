package infrastructure

import (
	"context"
	"fmt"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

var _ application.UserRepository = (*PostgresUserRepository)(nil)

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, username, name, role, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, user.Email, user.Username, user.Name, string(user.Role), user.PasswordHash).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	var role string

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, username, name, role, password_hash FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Username, &u.Name, &role, &u.PasswordHash)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	u.Role = domain.Role(role)
	return &u, nil
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	var role string

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, username, name, role, password_hash FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Username, &u.Name, &role, &u.PasswordHash)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	u.Role = domain.Role(role)
	return &u, nil
}

func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var u domain.User
	var role string

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, username, name, role, password_hash FROM users WHERE username = $1
	`, username).Scan(&u.ID, &u.Email, &u.Username, &u.Name, &role, &u.PasswordHash)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}

	u.Role = domain.Role(role)
	return &u, nil
}

func (r *PostgresUserRepository) AssignGateOperator(ctx context.Context, userID, eventID, assignedBy string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO gate_operator_assignments (user_id, event_id, assigned_by, status)
		VALUES ($1, $2, $3, 'ACTIVE')
		ON CONFLICT (user_id, event_id) DO UPDATE SET
			status = 'ACTIVE',
			assigned_by = EXCLUDED.assigned_by,
			assigned_at = NOW()
	`, userID, eventID, assignedBy)
	if err != nil {
		return fmt.Errorf("assign gate operator: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) ListAssignedGateOperators(ctx context.Context, eventID string) ([]application.AssignedOperator, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT goa.user_id, u.username, u.name, u.email, goa.assigned_at, goa.assigned_by, goa.status
		FROM gate_operator_assignments goa
		JOIN users u ON u.id = goa.user_id
		WHERE goa.event_id = $1 AND goa.status = 'ACTIVE'
		ORDER BY goa.assigned_at DESC
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list assigned gate operators: %w", err)
	}
	defer rows.Close()

	var operators []application.AssignedOperator
	for rows.Next() {
		var op application.AssignedOperator
		if err := rows.Scan(&op.UserID, &op.Username, &op.Name, &op.Email, &op.AssignedAt, &op.AssignedBy, &op.Status); err != nil {
			return nil, fmt.Errorf("scan assigned operator: %w", err)
		}
		operators = append(operators, op)
	}
	return operators, nil
}

func (r *PostgresUserRepository) RemoveGateOperator(ctx context.Context, userID, eventID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE gate_operator_assignments
		SET status = 'REVOKED'
		WHERE user_id = $1 AND event_id = $2 AND status = 'ACTIVE'
	`, userID, eventID)
	if err != nil {
		return fmt.Errorf("remove gate operator: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *PostgresUserRepository) SearchGateOperators(ctx context.Context, query string) ([]domain.User, error) {
	pattern := "%" + query + "%"
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, username, name, role, password_hash
		FROM users
		WHERE role = 'GATE_OPERATOR'
		  AND (username ILIKE $1 OR name ILIKE $1)
		ORDER BY username
		LIMIT 20
	`, pattern)
	if err != nil {
		return nil, fmt.Errorf("search gate operators: %w", err)
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		var role string
		if err := rows.Scan(&u.ID, &u.Email, &u.Username, &u.Name, &role, &u.PasswordHash); err != nil {
			return nil, fmt.Errorf("scan gate operator: %w", err)
		}
		u.Role = domain.Role(role)
		users = append(users, u)
	}
	return users, nil
}

type EventOwnershipAdapter struct {
	pool *pgxpool.Pool
}

func NewEventOwnershipAdapter(pool *pgxpool.Pool) *EventOwnershipAdapter {
	return &EventOwnershipAdapter{pool: pool}
}

var _ application.EventOwnershipChecker = (*EventOwnershipAdapter)(nil)

func (a *EventOwnershipAdapter) GetEventOrganizerID(ctx context.Context, eventID string) (string, error) {
	var organizerID string
	err := a.pool.QueryRow(ctx, `
		SELECT organizer_id::text FROM events WHERE id = $1
	`, eventID).Scan(&organizerID)
	if err == pgx.ErrNoRows {
		return "", fmt.Errorf("event not found")
	}
	if err != nil {
		return "", fmt.Errorf("get event organizer: %w", err)
	}
	return organizerID, nil
}
