package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ebk-tech/be-booking-events/internal/inventory/application"
	"github.com/ebk-tech/be-booking-events/internal/inventory/delivery"
	"github.com/ebk-tech/be-booking-events/internal/inventory/infrastructure"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/ebk-tech/be-booking-events/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}

	// Dependency injection: domain tidak tahu siapa yang inject, Infrastructure tidak tahu siapa yang pakai.
	ticketRepo := infrastructure.NewPostgresTicketRepository(pool)
	holdUC := application.NewHoldTicketUseCase(ticketRepo)
	expireUC := application.NewExpireHeldTicketsUseCase(ticketRepo)

	worker := infrastructure.NewCronExpiryWorker(expireUC)
	go worker.Start(ctx)

	r := chi.NewRouter()

	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"), //The url pointing to API definition
	))

	r.Get("/", RootHandler)
	r.Get("/check-health", HealthCheckHandler(pool))

	h := delivery.NewInventoryHandler(holdUC)
	r.Post("/api/v1/tickets/hold", h.HoldTicket)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	slog.Info("server listening", "addr", ":8080")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
	}
}
