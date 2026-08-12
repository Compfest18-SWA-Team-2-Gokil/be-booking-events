package infrastructure_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/inventory/application"
	"github.com/ebk-tech/be-booking-events/internal/inventory/domain"
	"github.com/ebk-tech/be-booking-events/internal/inventory/infrastructure"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool adalah koneksi shared untuk semua integration test dalam package ini.
// Di-setup sekali oleh TestMain, nil jika TEST_DATABASE_URL tidak di-set.
var testPool *pgxpool.Pool

// TestMain mengelola lifecycle pool dan migrasi untuk integration test.
// Jalankan dengan: TEST_DATABASE_URL="postgres://..." go test ./internal/inventory/infrastructure/...
func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		fmt.Println("INFO: TEST_DATABASE_URL tidak di-set, integration test dilewati.")
		fmt.Println("      Jalankan: docker-compose up -d lalu set TEST_DATABASE_URL.")
		os.Exit(m.Run()) // test akan di-skip via requirePool()
	}

	ctx := context.Background()

	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		fmt.Printf("FATAL: invalid TEST_DATABASE_URL: %v\n", err)
		os.Exit(1)
	}
	// MaxConns tinggi agar concurrent test benar-benar paralel di level DB.
	cfg.MaxConns = 50

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		fmt.Printf("FATAL: gagal connect ke DB: %v\n", err)
		os.Exit(1)
	}

	migrationSQL, err := os.ReadFile("../../../migrations/000001_create_inventory_tables.up.sql")
	if err != nil {
		fmt.Printf("FATAL: gagal baca migration: %v\n", err)
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		// Abaikan error jika tabel sudah ada (idempoten untuk re-run).
		_ = err
	}

	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// requirePool mengembalikan pool dan skip test jika DB tidak tersedia.
func requirePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("set TEST_DATABASE_URL untuk menjalankan integration test")
	}
	return testPool
}

// resetDB membersihkan semua data antar test agar independent satu sama lain.
func resetDB(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"TRUNCATE ticket_units, ticket_types, events RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("resetDB gagal: %v", err)
	}
}

func seedEvent(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO events (name, date, location)
		VALUES ('Test Concert', NOW() + INTERVAL '30 days', 'Jakarta')
		RETURNING id
	`).Scan(&id)
	if err != nil {
		t.Fatalf("seedEvent: %v", err)
	}
	return id
}

func seedTicketType(t *testing.T, pool *pgxpool.Pool, eventID string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO ticket_types (event_id, name, price, kind, total_quota)
		VALUES ($1, 'Festival GA', 150000, 'GA', 10000)
		RETURNING id
	`, eventID).Scan(&id)
	if err != nil {
		t.Fatalf("seedTicketType: %v", err)
	}
	return id
}

func seedTicketUnits(t *testing.T, pool *pgxpool.Pool, typeID string, count int) {
	t.Helper()
	for i := range count {
		_, err := pool.Exec(context.Background(),
			"INSERT INTO ticket_units (ticket_type_id) VALUES ($1)", typeID)
		if err != nil {
			t.Fatalf("seedTicketUnits[%d]: %v", i, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Test cases
// ---------------------------------------------------------------------------

func TestPostgresTicketRepo_HoldAvailableUnits_HappyPath(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	typeID := seedTicketType(t, pool, eventID)
	seedTicketUnits(t, pool, typeID, 3)

	repo := infrastructure.NewPostgresTicketRepository(pool)
	result, err := repo.HoldAvailableUnits(
		context.Background(),
		[]application.HoldRequest{{TicketTypeID: typeID, Quantity: 2}},
		time.Now().Add(5*time.Minute),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 units, got %d", len(result))
	}
	for _, u := range result {
		if u.Status != domain.StatusHeld {
			t.Errorf("unit %s status = %s, want HELD", u.ID, u.Status)
		}
		if u.HeldUntil == nil || u.HeldUntil.Before(time.Now()) {
			t.Errorf("unit %s held_until tidak valid: %v", u.ID, u.HeldUntil)
		}
	}
}

func TestPostgresTicketRepo_HoldAvailableUnits_InsufficientInventory(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	typeID := seedTicketType(t, pool, eventID)
	seedTicketUnits(t, pool, typeID, 1) // hanya 1 tersedia, request 2

	repo := infrastructure.NewPostgresTicketRepository(pool)
	_, err := repo.HoldAvailableUnits(
		context.Background(),
		[]application.HoldRequest{{TicketTypeID: typeID, Quantity: 2}},
		time.Now().Add(5*time.Minute),
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestPostgresTicketRepo_HoldAvailableUnits_LazyExpiry memverifikasi bahwa unit HELD
// yang sudah kadaluarsa bisa direbut pembeli baru dalam transaksi yang sama.
func TestPostgresTicketRepo_HoldAvailableUnits_LazyExpiry(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	typeID := seedTicketType(t, pool, eventID)

	// Insert 1 unit dengan status HELD yang sudah expired.
	_, err := pool.Exec(context.Background(), `
		INSERT INTO ticket_units (ticket_type_id, status, held_until)
		VALUES ($1, 'HELD', NOW() - INTERVAL '1 minute')
	`, typeID)
	if err != nil {
		t.Fatal(err)
	}

	repo := infrastructure.NewPostgresTicketRepository(pool)
	result, err := repo.HoldAvailableUnits(
		context.Background(),
		[]application.HoldRequest{{TicketTypeID: typeID, Quantity: 1}},
		time.Now().Add(5*time.Minute),
	)

	if err != nil {
		t.Fatalf("lazy expiry harus reclaim unit expired: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(result))
	}
	if result[0].Status != domain.StatusHeld {
		t.Errorf("unit status = %s, want HELD", result[0].Status)
	}
}

// TestPostgresTicketRepo_HoldAvailableUnits_MultiType memverifikasi hold dari 2 TicketType
// dalam satu transaksi (All-or-Nothing).
func TestPostgresTicketRepo_HoldAvailableUnits_MultiType(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	festTypeID := seedTicketType(t, pool, eventID)
	vipTypeID := seedTicketType(t, pool, eventID)
	seedTicketUnits(t, pool, festTypeID, 2)
	seedTicketUnits(t, pool, vipTypeID, 1)

	repo := infrastructure.NewPostgresTicketRepository(pool)
	result, err := repo.HoldAvailableUnits(
		context.Background(),
		[]application.HoldRequest{
			{TicketTypeID: festTypeID, Quantity: 2},
			{TicketTypeID: vipTypeID, Quantity: 1},
		},
		time.Now().Add(5*time.Minute),
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 units total, got %d", len(result))
	}
}

// TestPostgresTicketRepo_HoldAvailableUnits_MultiType_AllOrNothing memverifikasi bahwa
// jika satu tipe tidak mencukupi, semua hold dibatalkan (rollback).
func TestPostgresTicketRepo_HoldAvailableUnits_MultiType_AllOrNothing(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	festTypeID := seedTicketType(t, pool, eventID)
	vipTypeID := seedTicketType(t, pool, eventID)
	seedTicketUnits(t, pool, festTypeID, 2)
	// VIP sengaja tidak di-seed → tidak ada unit tersedia

	repo := infrastructure.NewPostgresTicketRepository(pool)
	_, err := repo.HoldAvailableUnits(
		context.Background(),
		[]application.HoldRequest{
			{TicketTypeID: festTypeID, Quantity: 2},
			{TicketTypeID: vipTypeID, Quantity: 1},
		},
		time.Now().Add(5*time.Minute),
	)

	if err == nil {
		t.Fatal("expected error karena VIP habis, got nil")
	}

	// Verifikasi festival units masih AVAILABLE (bukan HELD) — transaksi harus rollback.
	var heldCount int
	pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ticket_units WHERE ticket_type_id = $1 AND status = 'HELD'",
		festTypeID,
	).Scan(&heldCount)

	if heldCount != 0 {
		t.Errorf("setelah rollback, %d festival unit masih HELD (seharusnya 0)", heldCount)
	}
}

// TestPostgresTicketRepo_AntiOversell adalah tes inti dari sistem ini:
// N goroutine berebut 1 kursi terakhir secara bersamaan → tepat 1 yang berhasil.
// Ini memvalidasi bahwa SELECT FOR UPDATE benar-benar mencegah oversell.
func TestPostgresTicketRepo_AntiOversell(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	typeID := seedTicketType(t, pool, eventID)
	seedTicketUnits(t, pool, typeID, 1) // hanya 1 kursi

	repo := infrastructure.NewPostgresTicketRepository(pool)

	const concurrentBuyers = 50
	var (
		successCount atomic.Int32
		wg           sync.WaitGroup
		startGun     = make(chan struct{})
	)

	wg.Add(concurrentBuyers)
	for range concurrentBuyers {
		go func() {
			defer wg.Done()
			<-startGun // tunggu semua goroutine siap, lalu serbu serentak

			_, err := repo.HoldAvailableUnits(
				context.Background(),
				[]application.HoldRequest{{TicketTypeID: typeID, Quantity: 1}},
				time.Now().Add(5*time.Minute),
			)
			if err == nil {
				successCount.Add(1)
			}
		}()
	}

	close(startGun) // lepas semua goroutine serentak
	wg.Wait()

	if got := successCount.Load(); got != 1 {
		t.Errorf("anti-oversell GAGAL: %d transaksi berhasil hold 1 kursi (harus tepat 1)", got)
	}

	// Double-check: pastikan tepat 1 unit berstatus HELD di database.
	var heldCount int
	pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ticket_units WHERE ticket_type_id = $1 AND status = 'HELD'", typeID,
	).Scan(&heldCount)

	if heldCount != 1 {
		t.Errorf("database state: %d unit HELD (harus tepat 1)", heldCount)
	}
}

func TestPostgresTicketRepo_ExpireHeldUnits(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	typeID := seedTicketType(t, pool, eventID)

	// 2 unit expired, 1 masih aktif, 1 confirmed (tidak boleh tersentuh)
	pool.Exec(context.Background(), `
		INSERT INTO ticket_units (ticket_type_id, status, held_until) VALUES
		($1, 'HELD', NOW() - INTERVAL '10 minutes'),
		($1, 'HELD', NOW() - INTERVAL '1 minute'),
		($1, 'HELD', NOW() + INTERVAL '4 minutes'),
		($1, 'CONFIRMED', NULL)
	`, typeID)

	repo := infrastructure.NewPostgresTicketRepository(pool)
	count, err := repo.ExpireHeldUnits(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expired count = %d, want 2", count)
	}

	// Verifikasi state akhir di DB.
	var available, held, confirmed int
	pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ticket_units WHERE ticket_type_id = $1 AND status = 'AVAILABLE'", typeID,
	).Scan(&available)
	pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ticket_units WHERE ticket_type_id = $1 AND status = 'HELD'", typeID,
	).Scan(&held)
	pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM ticket_units WHERE ticket_type_id = $1 AND status = 'CONFIRMED'", typeID,
	).Scan(&confirmed)

	if available != 2 {
		t.Errorf("AVAILABLE count = %d, want 2", available)
	}
	if held != 1 {
		t.Errorf("HELD count = %d, want 1 (yang masih aktif)", held)
	}
	if confirmed != 1 {
		t.Errorf("CONFIRMED count = %d, want 1 (tidak boleh tersentuh)", confirmed)
	}
}

func TestPostgresTicketRepo_UpdateStatus(t *testing.T) {
	pool := requirePool(t)
	resetDB(t, pool)

	eventID := seedEvent(t, pool)
	typeID := seedTicketType(t, pool, eventID)

	// Seed 1 unit dengan status HELD
	var unitID string
	pool.QueryRow(context.Background(), `
		INSERT INTO ticket_units (ticket_type_id, status, held_until)
		VALUES ($1, 'HELD', NOW() + INTERVAL '5 minutes')
		RETURNING id
	`, typeID).Scan(&unitID)

	// Seed user and order for foreign key constraint
	var userID string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO users (email, name, role, password_hash)
		VALUES ('testbuyer@example.com', 'Test Buyer', 'BUYER', 'hash')
		RETURNING id
	`).Scan(&userID)
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}

	var orderID string
	err = pool.QueryRow(context.Background(), `
		INSERT INTO orders (buyer_id, event_id, total_amount)
		VALUES ($1, $2, 150000)
		RETURNING id
	`, userID, eventID).Scan(&orderID)
	if err != nil {
		t.Fatalf("seedOrder: %v", err)
	}

	repo := infrastructure.NewPostgresTicketRepository(pool)
	err = repo.UpdateStatus(context.Background(), unitID, domain.StatusPaymentPending, &orderID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verifikasi di DB.
	var status, gotOrderID string
	pool.QueryRow(context.Background(),
		"SELECT status, order_id FROM ticket_units WHERE id = $1", unitID,
	).Scan(&status, &gotOrderID)

	if status != string(domain.StatusPaymentPending) {
		t.Errorf("status = %s, want PAYMENT_PENDING", status)
	}
	if gotOrderID != orderID {
		t.Errorf("order_id = %s, want %s", gotOrderID, orderID)
	}
}
