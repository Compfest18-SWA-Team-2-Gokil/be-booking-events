package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	authapp "github.com/ebk-tech/be-booking-events/internal/auth/application"
	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	authinfra "github.com/ebk-tech/be-booking-events/internal/auth/infrastructure"
	checkinapp "github.com/ebk-tech/be-booking-events/internal/checkin/application"
	checkindelivery "github.com/ebk-tech/be-booking-events/internal/checkin/delivery"
	checkininfra "github.com/ebk-tech/be-booking-events/internal/checkin/infrastructure"
	dashboardapp "github.com/ebk-tech/be-booking-events/internal/dashboard/application"
	dashboarddelivery "github.com/ebk-tech/be-booking-events/internal/dashboard/delivery"
	dashboardinfra "github.com/ebk-tech/be-booking-events/internal/dashboard/infrastructure"
	"github.com/ebk-tech/be-booking-events/internal/inventory/application"
	"github.com/ebk-tech/be-booking-events/internal/inventory/delivery"
	"github.com/ebk-tech/be-booking-events/internal/inventory/infrastructure"
	queueapp "github.com/ebk-tech/be-booking-events/internal/queue/application"
	queuedelivery "github.com/ebk-tech/be-booking-events/internal/queue/delivery"
	queueinfra "github.com/ebk-tech/be-booking-events/internal/queue/infrastructure"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbURL := mustEnv("DATABASE_URL")
	jwtSecret := mustEnv("JWT_SECRET_KEY")
	qrSecret := mustEnv("QR_SECRET_KEY")
	queueSecret := mustEnv("QUEUE_SECRET_KEY")
	redisURL := mustEnv("REDIS_URL")

	// --- Postgres ---
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

	// --- Redis ---
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "err", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Error("redis ping failed", "err", err)
		os.Exit(1)
	}

	// --- Auth module ---
	tokenProvider := authinfra.NewJWTTokenProvider(jwtSecret)
	passwordHasher := authinfra.NewBcryptPasswordHasher()
	userRepo := authinfra.NewPostgresUserRepository(pool)
	registerUC := authapp.NewRegisterUseCase(userRepo, passwordHasher)
	loginUC := authapp.NewLoginUseCase(userRepo, passwordHasher, tokenProvider)
	assignGateOpUC := authapp.NewAssignGateOperatorUseCase(userRepo)
	authHandler := authdelivery.NewAuthHandler(registerUC, loginUC, assignGateOpUC, userRepo)
	authMiddleware := authdelivery.AuthMiddleware(tokenProvider)
	requireBuyer := authdelivery.RequireRole("BUYER")
	requireOrganizer := authdelivery.RequireRole("ORGANIZER")
	requireGateOperator := authdelivery.RequireRole("GATE_OPERATOR")

	// --- Inventory module ---
	ticketRepo := infrastructure.NewPostgresTicketRepository(pool)
	holdUC := application.NewHoldTicketUseCase(ticketRepo)
	expireUC := application.NewExpireHeldTicketsUseCase(ticketRepo)
	inventoryWorker := infrastructure.NewCronExpiryWorker(expireUC)
	go inventoryWorker.Start(ctx)

	// --- Dashboard module ---
	metricsRepo := dashboardinfra.NewPostgresMetricsRepository(pool)
	metricsUC := dashboardapp.NewGetEventMetricsUseCase(metricsRepo)
	dashboardHandler := dashboarddelivery.NewDashboardHandler(metricsUC)

	// --- Check-in module ---
	qrSigner := checkininfra.NewHMACQRSigner(qrSecret)
	checkinRepo := checkininfra.NewPostgresCheckinRepository(pool)
	issueUC := checkinapp.NewIssueTicketUseCase(checkinRepo, qrSigner)
	scanUC := checkinapp.NewScanTicketUseCase(checkinRepo, qrSigner)
	checkinHandler := checkindelivery.NewCheckinHandler(issueUC, scanUC)

	// --- Queue module ---
	queueRepo := queueinfra.NewRedisQueueRepository(redisClient)
	tokenSigner := queueinfra.NewHMACTokenSigner(queueSecret)
	joinUC := queueapp.NewJoinQueueUseCase(queueRepo)
	releaseUC := queueapp.NewReleaseQueueUseCase(queueRepo, tokenSigner, 10)
	validateTokenUC := queueapp.NewValidateQueueTokenUseCase(tokenSigner)
	queueWorker := queueinfra.NewCronReleaseWorker(releaseUC)
	go queueWorker.Start(ctx)
	queueHandler := queuedelivery.NewQueueHandler(joinUC, validateTokenUC, queueRepo)

	// --- Router ---
	r := chi.NewRouter()

	// Public: Auth
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)

		// Auth
		r.Get("/api/v1/auth/me", authHandler.Me)

		// Inventory — BUYER only
		r.With(requireBuyer).Post("/api/v1/tickets/hold", delivery.NewInventoryHandler(holdUC).HoldTicket)

		// Dashboard — ORGANIZER only
		r.With(requireOrganizer).Get("/api/v1/events/{eventID}/metrics", dashboardHandler.GetEventMetrics)

		// Gate operator assignment — ORGANIZER only
		r.With(requireOrganizer).Post("/api/v1/events/{eventID}/gate-operators", authHandler.AssignGateOperator)

		// Check-in — issue: ORGANIZER, scan: GATE_OPERATOR
		r.With(requireOrganizer).Post("/api/v1/checkin/issue", checkinHandler.IssueTicket)
		r.With(requireGateOperator).Post("/api/v1/checkin/scan", checkinHandler.ScanTicket)

		// Queue — BUYER only
		r.With(requireBuyer).Post("/api/v1/events/{eventID}/queue/join", queueHandler.JoinQueue)
		r.With(requireBuyer).Get("/api/v1/events/{eventID}/queue/status", queueHandler.GetQueueStatus)

		// Queue token validate — any authenticated
		r.Post("/api/v1/queue/token/validate", queueHandler.ValidateToken)
	})

	// --- HTTP server ---
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

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("env var wajib tidak ditemukan", "key", key)
		os.Exit(1)
	}
	return v
}
