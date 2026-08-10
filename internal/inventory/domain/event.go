package domain

import "time"

type Event struct {
	ID       string
	Name     string
	Date     time.Time
	Location string
}
