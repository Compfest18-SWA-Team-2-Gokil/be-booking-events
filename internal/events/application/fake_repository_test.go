package application_test

import (
	"context"

	"github.com/ebk-tech/be-booking-events/internal/events/domain"
)

type fakeEventRepo struct {
	events      map[string]*domain.Event
	ticketTypes map[string]*domain.TicketType
	unitCounts  map[string]int // ticketTypeID → provisioned count
}

func newFakeEventRepo() *fakeEventRepo {
	return &fakeEventRepo{
		events:      make(map[string]*domain.Event),
		ticketTypes: make(map[string]*domain.TicketType),
		unitCounts:  make(map[string]int),
	}
}

func (r *fakeEventRepo) CreateEvent(_ context.Context, e *domain.Event) error {
	e.ID = "event-" + e.Name
	r.events[e.ID] = e
	return nil
}

func (r *fakeEventRepo) ListEvents(_ context.Context) ([]*domain.Event, error) {
	result := make([]*domain.Event, 0, len(r.events))
	for _, e := range r.events {
		result = append(result, e)
	}
	return result, nil
}

func (r *fakeEventRepo) GetEvent(_ context.Context, id string) (*domain.Event, error) {
	e, ok := r.events[id]
	if !ok {
		return nil, domain.ErrEventNotFound
	}
	return e, nil
}

func (r *fakeEventRepo) CreateTicketType(_ context.Context, tt *domain.TicketType) error {
	tt.ID = "tt-" + tt.Name
	r.ticketTypes[tt.ID] = tt
	return nil
}

func (r *fakeEventRepo) ListTicketTypes(_ context.Context, eventID string) ([]*domain.TicketType, error) {
	var result []*domain.TicketType
	for _, tt := range r.ticketTypes {
		if tt.EventID == eventID {
			result = append(result, tt)
		}
	}
	return result, nil
}

func (r *fakeEventRepo) GetTicketType(_ context.Context, id string) (*domain.TicketType, error) {
	tt, ok := r.ticketTypes[id]
	if !ok {
		return nil, domain.ErrTicketTypeNotFound
	}
	return tt, nil
}

func (r *fakeEventRepo) ProvisionUnits(_ context.Context, ticketTypeID string, quantity int) error {
	r.unitCounts[ticketTypeID] += quantity
	return nil
}

func (r *fakeEventRepo) CountProvisionedUnits(_ context.Context, ticketTypeID string) (int, error) {
	return r.unitCounts[ticketTypeID], nil
}
