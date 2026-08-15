package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/audit/application"
	"github.com/ebk-tech/be-booking-events/internal/audit/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresAuditRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuditRepository(pool *pgxpool.Pool) *PostgresAuditRepository {
	return &PostgresAuditRepository{pool: pool}
}

var _ application.AuditRepository = (*PostgresAuditRepository)(nil)

func (r *PostgresAuditRepository) Insert(ctx context.Context, log *domain.AuditLog) error {
	metaJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`, log.ActorID, nullableStr(log.ActorRole), log.EntityName, log.EntityID, log.Action,
		nullableStr(log.FromState), nullableStr(log.ToState), metaJSON,
	).Scan(&log.ID, &log.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *PostgresAuditRepository) ListByEntity(ctx context.Context, entityName, entityID string) ([]*domain.AuditLog, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata, created_at
		FROM audit_logs
		WHERE entity_name = $1 AND entity_id = $2
		ORDER BY created_at ASC
	`, entityName, entityID)
	if err != nil {
		return nil, fmt.Errorf("query audit logs by entity: %w", err)
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

func (r *PostgresAuditRepository) ListRecent(ctx context.Context, limit int) ([]*domain.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, actor_id, actor_role, entity_name, entity_id, action, from_state, to_state, metadata, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent audit logs: %w", err)
	}
	defer rows.Close()

	return scanAuditLogs(rows)
}

func scanAuditLogs(rows any) ([]*domain.AuditLog, error) {
	type pgxRows interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	}
	r, ok := rows.(pgxRows)
	if !ok {
		return nil, fmt.Errorf("invalid rows")
	}

	var logs []*domain.AuditLog
	for r.Next() {
		log := &domain.AuditLog{}
		var actorRole, fromState, toState *string
		var metaRaw []byte
		if err := r.Scan(
			&log.ID, &log.ActorID, &actorRole, &log.EntityName, &log.EntityID,
			&log.Action, &fromState, &toState, &metaRaw, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if actorRole != nil {
			log.ActorRole = *actorRole
		}
		if fromState != nil {
			log.FromState = *fromState
		}
		if toState != nil {
			log.ToState = *toState
		}
		if len(metaRaw) > 0 {
			_ = json.Unmarshal(metaRaw, &log.Metadata)
		}
		logs = append(logs, log)
	}
	return logs, r.Err()
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
