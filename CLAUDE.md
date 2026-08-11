# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run server (requires DATABASE_URL env var)
source .env && go run ./cmd/server

# Run unit tests (no Docker needed)
go test ./internal/inventory/domain/... ./internal/inventory/application/...

# Run a single test
go test ./internal/inventory/application/... -run TestHoldTicketUseCase_Execute_SingleType_Success -v

# Run integration tests (requires postgres_test container)
docker-compose up -d postgres_test
TEST_DATABASE_URL="postgres://dev:dev@localhost:5433/booking_events_test" \
  go test ./internal/inventory/infrastructure/... -v

# Run all tests
docker-compose up -d
TEST_DATABASE_URL="postgres://dev:dev@localhost:5433/booking_events_test" go test ./... -v

# Apply migration
psql $DATABASE_URL -f migrations/000001_create_inventory_tables.up.sql
```

## Architecture

Clean Architecture — dependency hanya boleh mengarah ke dalam (delivery → application → domain ← infrastructure):

```
internal/inventory/
├── domain/          # Zero external deps. TicketUnit status machine + transition rules.
├── application/     # Use cases + TicketUnitRepository interface (ports.go).
├── infrastructure/  # PostgresTicketRepository, CronExpiryWorker.
└── delivery/        # chi HTTP handler.
```

**Key invariant:** `application/` hanya tahu `TicketUnitRepository` interface dari `ports.go` — tidak pernah import `infrastructure/`. Ini dijaga oleh compile-time check di `postgres_ticket_repository.go`:
```go
var _ application.TicketUnitRepository = (*PostgresTicketRepository)(nil)
```

## Anti-Oversell Engine

Inti sistem ada di `HoldAvailableUnits` (`infrastructure/postgres_ticket_repository.go`):

1. **Sort requests by TicketTypeID ASC** sebelum masuk transaksi — mencegah deadlock saat multi-type hold.
2. **Lazy expiry** dalam transaksi yang sama: unit HELD yang sudah melewati `held_until` di-reset ke AVAILABLE sebelum SELECT FOR UPDATE dijalankan.
3. **`SELECT FOR UPDATE ORDER BY id ASC LIMIT N`** per tipe — pessimistic lock, bukan optimistic.
4. **All-or-nothing**: jika satu tipe kurang stok, langsung return `domain.ErrTicketNotAvailable` dan transaksi di-rollback via `defer tx.Rollback`.

Background cleanup (`CronExpiryWorker`) jalan setiap 30 detik dengan `FOR UPDATE SKIP LOCKED` agar tidak memblok transaksi checkout aktif.

## Testing Strategy

- **`domain/` dan `application/`** — unit test murni, pakai `FakeRepository` dari `application/fake_repository_test.go`. Tidak butuh DB.
- **`infrastructure/`** — integration test, butuh Postgres sungguhan via `TEST_DATABASE_URL`. Test utama: `TestPostgresTicketRepo_AntiOversell` (50 goroutine berebut 1 kursi, harus tepat 1 yang berhasil).
- `TestMain` di `infrastructure/` otomatis skip semua test jika `TEST_DATABASE_URL` tidak di-set.

## Key Design Decisions

- **Hold duration 5 menit fixed** (`application.HoldDuration`) — tidak configurable per-event di MVP.
- **Status tiket** (`domain/ticket_unit.go`) menggunakan `TEXT + CHECK constraint` di Postgres, bukan ENUM — agar pgx v5 tidak perlu registrasi tipe kustom.
- **Tidak ada Redis** — hold state hanya di Postgres; cukup untuk MVP.
- **Partial indexes** di `ticket_units`: `idx_ticket_units_available` (WHERE status = 'AVAILABLE') untuk hot path anti-oversell, `idx_ticket_units_held_expired` (WHERE status = 'HELD') untuk cleanup worker.
