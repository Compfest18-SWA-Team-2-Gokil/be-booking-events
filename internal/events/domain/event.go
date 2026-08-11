package domain

import "time"

type Kind string

const (
	KindGA     Kind = "GA"
	KindSeated Kind = "SEATED"
)

type Event struct {
	ID          string    `json:"id"`
	OrganizerID string    `json:"organizer_id"`
	Name        string    `json:"name"`
	Date        time.Time `json:"date"`
	Location    string    `json:"location"`
}

func (e *Event) Validate() error {
	if e.Name == "" {
		return ErrNameRequired
	}
	if e.Location == "" {
		return ErrLocationRequired
	}
	if !e.Date.After(time.Now()) {
		return ErrDateMustBeFuture
	}
	return nil
}

type TicketType struct {
	ID         string `json:"id"`
	EventID    string `json:"event_id"`
	Name       string `json:"name"`
	Price      int64  `json:"price"`
	Kind       Kind   `json:"kind"`
	TotalQuota int    `json:"total_quota"`
}

func (tt *TicketType) Validate() error {
	if tt.Name == "" {
		return ErrNameRequired
	}
	if tt.Price < 0 {
		return ErrInvalidPrice
	}
	if tt.TotalQuota < 1 {
		return ErrInvalidQuota
	}
	if tt.Kind != KindGA && tt.Kind != KindSeated {
		return ErrInvalidKind
	}
	return nil
}
