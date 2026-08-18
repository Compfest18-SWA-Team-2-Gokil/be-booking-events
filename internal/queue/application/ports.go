package application

import (
	"context"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/domain"
)

// QueueRepository mengelola state antrean di Redis.
type QueueRepository interface {
	// Join mendaftarkan user ke antrean. Mengembalikan posisi (0-based).
	// NX semantics: jika user sudah ada, kembalikan posisi existing.
	Join(ctx context.Context, eventID, userID string) (position int64, err error)

	// Position mengambil posisi user saat ini. -1 jika user tidak ada di antrean.
	Position(ctx context.Context, eventID, userID string) (int64, error)

	// PopTop mengambil dan menghapus N user teratas dari antrean (FIFO by score).
	PopTop(ctx context.Context, eventID string, n int64) (userIDs []string, err error)

	// QueueSize mengembalikan jumlah user yang masih dalam antrean.
	QueueSize(ctx context.Context, eventID string) (int64, error)

	// SetToken menyimpan queue token untuk user (untuk diambil via polling).
	SetToken(ctx context.Context, eventID, userID, tokenStr string, ttl time.Duration) error

	// GetToken mengambil queue token yang sudah di-generate untuk user.
	// Mengembalikan "" jika belum tersedia.
	GetToken(ctx context.Context, eventID, userID string) (string, error)

	// AddActiveQueue mendaftarkan eventID ke set antrean aktif.
	AddActiveQueue(ctx context.Context, eventID string) error

	// GetActiveQueues mengambil semua eventID yang antreannya aktif.
	GetActiveQueues(ctx context.Context) ([]string, error)

	// RemoveActiveQueue menghapus eventID dari set antrean aktif.
	RemoveActiveQueue(ctx context.Context, eventID string) error

	// IncrRequestRate menambah counter request/detik untuk event ini.
	// Mengembalikan jumlah request dalam window 1 detik saat ini.
	IncrRequestRate(ctx context.Context, eventID string) (int64, error)
}

// TokenSigner menangani signing dan verifikasi queue token.
type TokenSigner interface {
	Sign(token domain.QueueToken) (string, error)
	Verify(tokenStr string) (*domain.QueueToken, error)
}
