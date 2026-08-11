package infrastructure

import (
	"context"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/auth/application"
	"github.com/ebk-tech/be-booking-events/internal/auth/domain"
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
		INSERT INTO users (email, name, role, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, user.Email, user.Name, string(user.Role), user.PasswordHash).Scan(&user.ID)

	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	var role string

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, name, role, password_hash FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.Name, &role, &u.PasswordHash)

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
		SELECT id, email, name, role, password_hash FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.Name, &role, &u.PasswordHash)

	if err == pgx.ErrNoRows {
		return nil, domain.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	u.Role = domain.Role(role)
	return &u, nil
}

func (r *PostgresUserRepository) AssignGateOperator(ctx context.Context, userID, eventID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO gate_operator_assignments (user_id, event_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, event_id) DO NOTHING
	`, userID, eventID)
	if err != nil {
		return fmt.Errorf("assign gate operator: %w", err)
	}
	return nil
}
