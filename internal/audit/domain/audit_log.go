package domain

import "time"

type AuditLog struct {
	ID         string         `json:"id"`
	ActorID    *string        `json:"actor_id,omitempty"`
	ActorRole  string         `json:"actor_role,omitempty"`
	EntityName string         `json:"entity_name"`
	EntityID   string         `json:"entity_id"`
	Action     string         `json:"action"`
	FromState  string         `json:"from_state,omitempty"`
	ToState    string         `json:"to_state,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}
