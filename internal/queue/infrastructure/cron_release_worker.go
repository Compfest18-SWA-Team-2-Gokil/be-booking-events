package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/queue/application"
)

const releaseInterval = 5 * time.Second

type CronReleaseWorker struct {
	usecase  *application.ReleaseQueueUseCase
	interval time.Duration
}

func NewCronReleaseWorker(usecase *application.ReleaseQueueUseCase) *CronReleaseWorker {
	return &CronReleaseWorker{usecase: usecase, interval: releaseInterval}
}

// Start menjalankan loop release antrean sampai ctx dibatalkan. Panggil dalam goroutine terpisah.
func (w *CronReleaseWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.usecase.ExecuteAll(ctx); err != nil {
				slog.Error("release queue worker error", "err", err)
			}
		}
	}
}
