package domain

import "time"

// QueueToken adalah bukti bahwa user telah lulus antrean dan berhak mengakses checkout.
type QueueToken struct {
	UserID    string    `json:"user_id"`
	EventID   string    `json:"event_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (t *QueueToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *QueueToken) IsValid() bool {
	return t.UserID != "" && t.EventID != "" && !t.IsExpired()
}
