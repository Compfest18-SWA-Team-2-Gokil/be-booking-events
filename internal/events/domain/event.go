package domain

import "time"

type Kind string

const (
	KindGA     Kind = "GA"
	KindSeated Kind = "SEATED"
)

type Category string

const (
	CategoryMusic    Category = "music"
	CategoryOlahraga Category = "olahraga"
	CategorySeni     Category = "seni"
	CategoryWorkshop Category = "workshop"
)

func ValidCategory(c Category) bool {
	switch c {
	case CategoryMusic, CategoryOlahraga, CategorySeni, CategoryWorkshop:
		return true
	}
	return false
}

type Event struct {
	ID          string    `json:"id"`
	OrganizerID string    `json:"organizer_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    Category  `json:"category"`
	Date        time.Time `json:"date"`
	Location    string    `json:"location"`
	ImageURL    string    `json:"image_url,omitempty"`
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
	if !ValidCategory(e.Category) {
		return ErrInvalidCategory
	}
	return nil
}

type TicketType struct {
	ID          string `json:"id"`
	EventID     string `json:"event_id"`
	Name        string `json:"name"`
	Price       int64  `json:"price"`
	Kind        Kind   `json:"kind"`
	TotalQuota  int    `json:"total_quota"`
	PriceStatus string `json:"price_status"`
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
