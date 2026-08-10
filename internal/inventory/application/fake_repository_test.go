package application_test

import (
	"context"
	"sync"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/inventory/application"
	"github.com/ebk-tech/be-booking-events/internal/inventory/domain"
)

// fakeRepo adalah implementasi in-memory dari TicketUnitRepository untuk unit test use case.
// Mutex membuatnya aman untuk concurrent test, tapi tidak menguji DATABASE locking —
// untuk itu pakai integration test di infrastructure layer.
type fakeRepo struct {
	mu    sync.Mutex
	units map[string]*domain.TicketUnit
}

func newFakeRepo(units ...*domain.TicketUnit) *fakeRepo {
	r := &fakeRepo{units: make(map[string]*domain.TicketUnit)}
	for _, u := range units {
		r.units[u.ID] = u
	}
	return r
}

func (r *fakeRepo) HoldAvailableUnits(
	_ context.Context,
	requests []application.HoldRequest,
	holdUntil time.Time,
) ([]*domain.TicketUnit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Kumpulkan dulu tanpa ubah state (simulasi atomicity transaksi DB).
	var toHold []*domain.TicketUnit
	for _, req := range requests {
		var found []*domain.TicketUnit
		for _, u := range r.units {
			if u.TicketTypeID != req.TicketTypeID || u.Status != domain.StatusAvailable {
				continue
			}
			alreadySelected := false
			for _, th := range toHold {
				if th.ID == u.ID {
					alreadySelected = true
					break
				}
			}
			if !alreadySelected {
				found = append(found, u)
			}
			if len(found) == req.Quantity {
				break
			}
		}
		if len(found) < req.Quantity {
			return nil, domain.ErrTicketNotAvailable
		}
		toHold = append(toHold, found...)
	}

	// Baru terapkan hold setelah semua tipe berhasil dikumpulkan.
	for _, u := range toHold {
		u.Status = domain.StatusHeld
		u.HeldUntil = &holdUntil
	}

	return toHold, nil
}

func (r *fakeRepo) UpdateStatus(
	_ context.Context,
	unitID string,
	newStatus domain.TicketUnitStatus,
	orderID *string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.units[unitID]
	if !ok {
		return domain.ErrTicketNotFound
	}
	u.Status = newStatus
	u.OrderID = orderID
	return nil
}

func (r *fakeRepo) ExpireHeldUnits(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, u := range r.units {
		if u.IsHoldExpired() {
			u.Status = domain.StatusAvailable
			u.HeldUntil = nil
			count++
		}
	}
	return count, nil
}

// Compile-time check.
var _ application.TicketUnitRepository = (*fakeRepo)(nil)
