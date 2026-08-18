package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/domain"
)

// HoldRequest merepresentasikan satu item dalam permintaan hold.
type HoldRequest struct {
	TicketTypeID string
	Quantity     int
}

// TicketUnitRepository adalah kontrak yang harus dipenuhi oleh Infrastructure layer.
// Application layer tidak tahu implementasi konkret (PostgreSQL, in-memory, dsb).
type TicketUnitRepository interface {
	// HoldAvailableUnits mengunci unit tiket dalam satu transaksi database.
	//
	// Jaminan anti-deadlock (Open Question #2 & #4 dari TDD):
	//   1. Requests diurutkan ascending by TicketTypeID sebelum diproses —
	//      semua transaksi akuisisi lock tipe dalam urutan yang sama.
	//   2. Dalam setiap tipe, unit dikunci ORDER BY id ASC —
	//      konsisten dengan transaksi lain yang mungkin lock subset baris yang sama.
	HoldAvailableUnits(ctx context.Context, requests []HoldRequest, holdUntil time.Time) ([]*domain.TicketUnit, error)

	// UpdateStatus memindahkan status sebuah unit ke status baru.
	// orderID diisi saat unit dikaitkan ke sebuah Order (PAYMENT_PENDING ke atas).
	UpdateStatus(ctx context.Context, unitID string, newStatus domain.TicketUnitStatus, orderID *string) error

	// ExpireHeldUnits merilis semua unit HELD yang sudah lewat held_until.
	// Dipanggil oleh background cleanup worker setiap 30 detik.
	ExpireHeldUnits(ctx context.Context) (int, error)
}
