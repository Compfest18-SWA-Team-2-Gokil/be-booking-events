// Package audit menyediakan append-only audit logger untuk transisi status tiket/order.
package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Logger struct {
	pool *pgxpool.Pool
}

func NewLogger(pool *pgxpool.Pool) *Logger {
	return &Logger{pool: pool}
}

type Entry struct {
	ActorID    string // UUID user yang melakukan aksi; kosong = sistem
	ActorRole  string // BUYER | ORGANIZER | GATE_OPERATOR | ADMIN | SYSTEM
	EntityType string // order | ticket_unit | payment
	EntityID   string // UUID entitas
	Action     string // CONFIRM_PAYMENT | LOST_SEAT | REFUND_REQUESTED | REFUND_APPROVED | ADMITTED | ADMIN_OVERRIDE | LOGOUT
	FromStatus string
	ToStatus   string
	Metadata   map[string]any // info tambahan (reason, ip, dll)
}

// Log menyisipkan satu baris ke audit_logs. Fire-and-forget: error tidak di-propagate ke caller.
func (l *Logger) Log(ctx context.Context, e Entry) {
	var meta []byte
	if len(e.Metadata) > 0 {
		meta, _ = json.Marshal(e.Metadata)
	}

	actorID := nullUUID(e.ActorID)
	_, _ = l.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, entity_type, entity_id, action, from_status, to_status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, actorID, e.ActorRole, e.EntityType, e.EntityID, e.Action,
		nullStr(e.FromStatus), nullStr(e.ToStatus), nullStr(string(meta)))
}

func nullUUID(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
