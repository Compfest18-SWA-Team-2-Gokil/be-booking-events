package domain

import "errors"

var (
	ErrTicketNotAvailable      = errors.New("no available ticket units")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrTicketNotFound          = errors.New("ticket unit not found")
)
