package application

import "context"

// TicketTypeMetrics adalah agregasi status tiket untuk satu TicketType.
// Sold = PAYMENT_PENDING + CONFIRMED (tiket yang sudah dibayar, belum/sudah masuk).
type TicketTypeMetrics struct {
	TicketTypeID string `json:"ticket_type_id"`
	Available    int    `json:"available"`
	Held         int    `json:"held"`
	Sold         int    `json:"sold"`
	Admitted     int    `json:"admitted"`
	Refunded     int    `json:"refunded"`
	Total        int    `json:"total"`
}

type MetricsRepository interface {
	GetEventMetrics(ctx context.Context, eventID string) ([]TicketTypeMetrics, error)
}
