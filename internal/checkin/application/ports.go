package application

import (
	"context"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/checkin/domain"
)

// QRSigner menangani signing dan verifikasi QR code tiket.
// Implementasi konkret: HMACQRSigner di infrastructure layer.
type QRSigner interface {
	Sign(payload domain.QRPayload) (string, error)
	Verify(qrContent string) (*domain.QRPayload, error)
}

// CheckinRepository adalah port untuk operasi database modul check-in.
// Read-only terhadap tabel yang dimiliki modul Inventory.
type CheckinRepository interface {
	// GetConfirmedUnit mengambil data tiket jika statusnya CONFIRMED.
	GetConfirmedUnit(ctx context.Context, ticketUnitID string) (*domain.CheckinTicket, error)

	// AdmitUnit melakukan atomic UPDATE: CONFIRMED → ADMITTED.
	// Mengembalikan ErrAlreadyAdmitted jika 0 baris ter-update.
	AdmitUnit(ctx context.Context, ticketUnitID, gateOperatorID string) error

	// IsGateOperatorAssigned memvalidasi bahwa gate operator terdaftar untuk event ini.
	IsGateOperatorAssigned(ctx context.Context, gateOperatorID, eventID string) (bool, error)
}
