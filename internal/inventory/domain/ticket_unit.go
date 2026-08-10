package domain

import "time"

type TicketUnitStatus string

const (
	StatusAvailable      TicketUnitStatus = "AVAILABLE"
	StatusHeld           TicketUnitStatus = "HELD"
	StatusPaymentPending TicketUnitStatus = "PAYMENT_PENDING"
	StatusConfirmed      TicketUnitStatus = "CONFIRMED"
	StatusRefunded       TicketUnitStatus = "REFUNDED"
	StatusAdmitted       TicketUnitStatus = "ADMITTED"
)

var validTransitions = map[TicketUnitStatus][]TicketUnitStatus{
	StatusAvailable:      {StatusHeld},
	StatusHeld:           {StatusAvailable, StatusPaymentPending},
	StatusPaymentPending: {StatusConfirmed, StatusRefunded},
	StatusConfirmed:      {StatusRefunded, StatusAdmitted},
	StatusRefunded:       {StatusAvailable},
	StatusAdmitted:       {},
}

type TicketUnit struct {
	ID           string
	TicketTypeID string
	SeatLabel    *string // nil for GA tickets
	Status       TicketUnitStatus
	HeldUntil    *time.Time
	OrderID      *string
}

func (t *TicketUnit) CanTransitionTo(next TicketUnitStatus) bool {
	for _, allowed := range validTransitions[t.Status] {
		if allowed == next {
			return true
		}
	}
	return false
}

func (t *TicketUnit) IsHoldExpired() bool {
	return t.Status == StatusHeld && t.HeldUntil != nil && time.Now().After(*t.HeldUntil)
}
