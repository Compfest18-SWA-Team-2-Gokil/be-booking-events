package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/application"
)

// expiryInterval: 30 detik sesuai TDD Bab 4 (active cleanup job).
const expiryInterval = 30 * time.Second

type CronExpiryWorker struct {
	usecase  *application.ExpireHeldTicketsUseCase
	interval time.Duration
}

func NewCronExpiryWorker(usecase *application.ExpireHeldTicketsUseCase) *CronExpiryWorker {
	return &CronExpiryWorker{
		usecase:  usecase,
		interval: expiryInterval,
	}
}

// Start menjalankan loop expiry sampai ctx dibatalkan. Panggil dalam goroutine terpisah.
func (w *CronExpiryWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():

			return
		case <-ticker.C:
			if err := w.usecase.Execute(ctx); err != nil {
				slog.Error("expiry worker error", "err", err)
			}
		}
	}
}
