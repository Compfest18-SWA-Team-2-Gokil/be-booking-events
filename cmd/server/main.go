package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	adminapp "github.com/ebk-tech/be-booking-events/internal/admin/application"
	admindelivery "github.com/ebk-tech/be-booking-events/internal/admin/delivery"
	admininfra "github.com/ebk-tech/be-booking-events/internal/admin/infrastructure"
	authapp "github.com/ebk-tech/be-booking-events/internal/auth/application"
	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	authinfra "github.com/ebk-tech/be-booking-events/internal/auth/infrastructure"
	checkinapp "github.com/ebk-tech/be-booking-events/internal/checkin/application"
	checkindelivery "github.com/ebk-tech/be-booking-events/internal/checkin/delivery"
	checkininfra "github.com/ebk-tech/be-booking-events/internal/checkin/infrastructure"
	dashboardapp "github.com/ebk-tech/be-booking-events/internal/dashboard/application"
	dashboarddelivery "github.com/ebk-tech/be-booking-events/internal/dashboard/delivery"
	dashboardinfra "github.com/ebk-tech/be-booking-events/internal/dashboard/infrastructure"
	eventsapp "github.com/ebk-tech/be-booking-events/internal/events/application"
	eventsdelivery "github.com/ebk-tech/be-booking-events/internal/events/delivery"
	eventsinfra "github.com/ebk-tech/be-booking-events/internal/events/infrastructure"
	inventoryapp "github.com/ebk-tech/be-booking-events/internal/inventory/application"
	inventorydelivery "github.com/ebk-tech/be-booking-events/internal/inventory/delivery"
	inventoryinfra "github.com/ebk-tech/be-booking-events/internal/inventory/infrastructure"
	ordersapp "github.com/ebk-tech/be-booking-events/internal/orders/application"
	ordersdelivery "github.com/ebk-tech/be-booking-events/internal/orders/delivery"
	ordersinfra "github.com/ebk-tech/be-booking-events/internal/orders/infrastructure"
	queueapp "github.com/ebk-tech/be-booking-events/internal/queue/application"
	queuedelivery "github.com/ebk-tech/be-booking-events/internal/queue/delivery"
	queueinfra "github.com/ebk-tech/be-booking-events/internal/queue/infrastructure"
	"strconv"
	"github.com/ebk-tech/be-booking-events/cmd/server/routes"
	"github.com/ebk-tech/be-booking-events/internal/migration"
	appmiddleware "github.com/ebk-tech/be-booking-events/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Attempt to load .env file, but ignore if it doesn't exist
	_ = godotenv.Load()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbURL := mustEnv("DATABASE_URL")
	jwtSecret := mustEnv("JWT_SECRET_KEY")
	xenditSecretKey := mustEnv("XENDIT_SECRET_KEY")
	xenditCallbackToken := mustEnv("XENDIT_CALLBACK_TOKEN")
	minioEndpoint := mustEnv("MINIO_ENDPOINT")
	minioAccessKey := mustEnv("MINIO_ACCESS_KEY")
	minioSecretKey := mustEnv("MINIO_SECRET_KEY")
	minioBucket := mustEnv("MINIO_BUCKET")
	minioPublicURL := mustEnv("MINIO_PUBLIC_URL")
	minioUseSSL, _ := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))
	qrSecret := mustEnv("QR_SECRET_KEY")
	queueSecret := mustEnv("QUEUE_SECRET_KEY")
	redisURL := mustEnv("REDIS_URL")

	// --- Postgres ---
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		slog.Error("failed to parse database URL", "err", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = 50 // cukup untuk 1000 concurrent req (DB bottleneck di lock, bukan koneksi)
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		slog.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "err", err)
		os.Exit(1)
	}

	// --- Run Auto-Migration ---
	slog.Info("running database auto-migration...")
	dbConn, err := pool.Acquire(ctx)
	if err != nil {
		slog.Error("failed to acquire db connection for migration", "err", err)
		os.Exit(1)
	}
	
	if err := migration.EnsureLogTable(ctx, dbConn.Conn()); err != nil {
		slog.Error("failed to ensure migration log table", "err", err)
		dbConn.Release()
		os.Exit(1)
	}

	if err := migration.RunUp(ctx, dbConn.Conn()); err != nil {
		slog.Error("database auto-migration failed", "err", err)
		dbConn.Release()
		os.Exit(1)
	}
	dbConn.Release()
	slog.Info("database auto-migration completed successfully")

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

	// --- Main page + health check
	
	// --- Auth module ---
	tokenProvider := authinfra.NewJWTTokenProvider(jwtSecret)
	passwordHasher := authinfra.NewBcryptPasswordHasher()
	userRepo := authinfra.NewPostgresUserRepository(pool)
	registerUC := authapp.NewRegisterUseCase(userRepo, passwordHasher)
	loginUC := authapp.NewLoginUseCase(userRepo, passwordHasher, tokenProvider)
	assignGateOpUC := authapp.NewAssignGateOperatorUseCase(userRepo)
	authHandler := authdelivery.NewAuthHandler(registerUC, loginUC, assignGateOpUC, userRepo)

	// --- Inventory module ---
	ticketRepo := inventoryinfra.NewPostgresTicketRepository(pool)
	holdUC := inventoryapp.NewHoldTicketUseCase(ticketRepo)
	expireUC := inventoryapp.NewExpireHeldTicketsUseCase(ticketRepo)
	inventoryWorker := inventoryinfra.NewCronExpiryWorker(expireUC)
	go inventoryWorker.Start(ctx)
	inventoryHandler := inventorydelivery.NewInventoryHandler(holdUC)

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

	// --- Orders module ---
	orderRepo := ordersinfra.NewPostgresOrderRepository(pool)
	xenditProvider := ordersinfra.NewXenditPaymentProvider(xenditSecretKey)
	createOrderUC := ordersapp.NewCreateOrderUseCase(orderRepo)
	initiatePayUC := ordersapp.NewInitiatePaymentUseCase(orderRepo, xenditProvider)
	confirmPayUC := ordersapp.NewConfirmPaymentUseCase(orderRepo)
	requestRefundUC := ordersapp.NewRequestRefundUseCase(orderRepo)
	approveRefundUC := ordersapp.NewApproveRefundUseCase(orderRepo, xenditProvider)
	getOrderUC := ordersapp.NewGetOrderUseCase(orderRepo)
	ordersHandler := ordersdelivery.NewOrdersHandler(
		createOrderUC, initiatePayUC, confirmPayUC,
		requestRefundUC, approveRefundUC, getOrderUC,
		xenditCallbackToken,
	)

	// --- MinIO ---
	minioStorage, err := eventsinfra.NewMinIOStorageProvider(
		minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioPublicURL, minioUseSSL,
	)
	if err != nil {
		slog.Error("gagal koneksi ke MinIO", "err", err)
		os.Exit(1)
	}
	if err := minioStorage.EnsureBucket(ctx); err != nil {
		slog.Error("gagal setup MinIO bucket", "err", err)
		os.Exit(1)
	}

	// --- Events module ---
	eventRepo := eventsinfra.NewPostgresEventRepository(pool)
	createEventUC := eventsapp.NewCreateEventUseCase(eventRepo)
	getEventUC := eventsapp.NewGetEventUseCase(eventRepo)
	listEventsUC := eventsapp.NewListEventsUseCase(eventRepo)
	updateEventUC := eventsapp.NewUpdateEventUseCase(eventRepo)
	deleteEventUC := eventsapp.NewDeleteEventUseCase(eventRepo)
	createTicketTypeUC := eventsapp.NewCreateTicketTypeUseCase(eventRepo)
	listTicketTypesUC := eventsapp.NewListTicketTypesUseCase(eventRepo)
	updateTicketTypeUC := eventsapp.NewUpdateTicketTypeUseCase(eventRepo)
	deleteTicketTypeUC := eventsapp.NewDeleteTicketTypeUseCase(eventRepo)
	provisionUnitsUC := eventsapp.NewProvisionUnitsUseCase(eventRepo)
	uploadImageUC := eventsapp.NewUploadEventImageUseCase(eventRepo, minioStorage)
	eventsHandler := eventsdelivery.NewEventsHandler(
		createEventUC, getEventUC, listEventsUC, updateEventUC, deleteEventUC,
		createTicketTypeUC, listTicketTypesUC, updateTicketTypeUC, deleteTicketTypeUC,
		provisionUnitsUC, uploadImageUC,
	)

	// --- Admin module ---
	adminRepo := admininfra.NewPostgresAdminRepository(pool)
	adminUC := adminapp.NewAdminUseCases(adminRepo)
	adminHandler := admindelivery.NewAdminHandler(adminUC)

	// --- Router ---
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})


	routes.Register(r, routes.Deps{
		AuthMiddleware:      authdelivery.AuthMiddleware(tokenProvider),
		RequireBuyer:        authdelivery.RequireRole("BUYER"),
		RequireOrganizer:    authdelivery.RequireRole("ORGANIZER"),
		RequireGateOperator: authdelivery.RequireRole("GATE_OPERATOR"),
		RequireAdmin:        authdelivery.RequireRole("ADMIN"),
		RequireQueueToken:   appmiddleware.RequireQueueToken(validateTokenUC),
		Idempotency:         appmiddleware.Idempotency(redisClient),

		Auth:      authHandler,
		Inventory: inventoryHandler,
		Dashboard: dashboardHandler,
		Checkin:   checkinHandler,
		Queue:     queueHandler,
		Events:    eventsHandler,
		Orders:    ordersHandler,
		Admin:     adminHandler,
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
