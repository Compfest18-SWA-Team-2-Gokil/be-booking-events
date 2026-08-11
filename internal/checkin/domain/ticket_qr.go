package domain

import "time"

// QRPayload adalah isi data yang di-encode ke dalam QR code tiket.
type QRPayload struct {
	TicketUnitID string    `json:"ticket_unit_id"`
	OrderID      string    `json:"order_id"`
	EventID      string    `json:"event_id"`
	IssuedAt     time.Time `json:"issued_at"`
}

func (p *QRPayload) Validate() error {
	if p.TicketUnitID == "" || p.EventID == "" {
		return ErrInvalidQRPayload
	}
	return nil
}

// CheckinTicket adalah read-model tiket yang siap di-admit (status CONFIRMED).
type CheckinTicket struct {
	ID      string
	OrderID string
	EventID string
}
