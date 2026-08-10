# be-booking-events

Backend service untuk sistem ticketing event — mengelola inventori tiket dengan mekanisme anti-oversell berbasis pessimistic locking.

## Stack

- **Go 1.26** — runtime
- **PostgreSQL 17** — primary database
- **pgx v5** — driver Postgres
- **chi v5** — HTTP router
- **Docker Compose** — local dev & test database

## Arsitektur

Clean Architecture dengan 3 layer:

```
internal/inventory/
├── domain/          # Business logic murni, zero dependency external
├── application/     # Use cases, orchestrates domain + ports
├── infrastructure/  # Postgres repository, cron worker
└── delivery/        # HTTP handler (chi)
```

Dependency flow: `delivery → application → domain ← infrastructure`

## Cara Menjalankan

### 1. Prasyarat

- Go 1.22+
- Docker & Docker Compose

### 2. Setup environment

```bash
cp .env.example .env
```

Edit `.env` sesuai kebutuhan (default sudah cocok untuk Docker Compose lokal).

### 3. Jalankan database

```bash
docker-compose up -d postgres
```

### 4. Jalankan server

```bash
source .env && go run ./cmd/server
```

Server berjalan di `http://localhost:8080`.

## Migrasi Database

Migrasi dijalankan manual (belum ada migration runner otomatis):

```bash
# Up
psql $DATABASE_URL -f migrations/000001_create_inventory_tables.up.sql

# Down
psql $DATABASE_URL -f migrations/000001_create_inventory_tables.down.sql
```

## API

### `POST /api/v1/tickets/hold`

Memesan hold tiket (berlaku 5 menit). Mendukung multi-type dalam satu transaksi (all-or-nothing).

**Request:**
```json
{
  "items": [
    { "ticket_type_id": "uuid-festival-ga", "quantity": 2 },
    { "ticket_type_id": "uuid-vip", "quantity": 1 }
  ]
}
```

**Response `200 OK`:**
```json
{
  "held_until": "2026-08-10T20:00:00+07:00",
  "unit_ids": ["uuid-1", "uuid-2", "uuid-3"]
}
```

**Response `409 Conflict`** — stok tidak mencukupi:
```json
{ "error": "tiket tidak tersedia" }
```

## Testing

### Unit test (tanpa Docker)

Menguji domain rules dan use cases menggunakan fake repository in-memory.

```bash
go test ./internal/inventory/domain/... ./internal/inventory/application/...
```

### Integration test (butuh Docker)

Menguji Postgres repository secara langsung, termasuk **anti-oversell** (50 goroutine berebut 1 kursi).

```bash
# Jalankan database test
docker-compose up -d postgres_test

# Tunggu sehat, lalu jalankan test
TEST_DATABASE_URL="postgres://dev:dev@localhost:5433/booking_events_test" \
  go test ./internal/inventory/infrastructure/... -v
```

### Semua test sekaligus

```bash
docker-compose up -d

TEST_DATABASE_URL="postgres://dev:dev@localhost:5433/booking_events_test" \
  go test ./... -v
```

## Skema Database

| Tabel | Deskripsi |
|---|---|
| `events` | Data event (nama, tanggal, lokasi) |
| `ticket_types` | Tipe tiket per event (GA/Seated, kuota, harga) |
| `ticket_units` | Satu baris = satu tiket fisik; status machine: `AVAILABLE → HELD → PAYMENT_PENDING → CONFIRMED → ADMITTED / REFUNDED` |

### Status Tiket

```
AVAILABLE → HELD → PAYMENT_PENDING → CONFIRMED → ADMITTED
                ↘                 ↘              ↘
              AVAILABLE          AVAILABLE      REFUNDED
```

## Anti-Oversell Engine

Kunci utama ada di `postgres_ticket_repository.go`:

1. **`SELECT FOR UPDATE SKIP LOCKED`** — mengunci baris AVAILABLE secara pesimistik; transaksi concurrent yang kalah tidak memblok, langsung lanjut ke unit lain.
2. **Lazy expiry** — saat query AVAILABLE, unit HELD yang sudah melewati `held_until` diperlakukan sebagai AVAILABLE dalam transaksi yang sama.
3. **All-or-nothing** — jika salah satu tipe kurang stok, seluruh transaksi di-rollback.
4. **Cron worker** — cleanup background setiap menit untuk reset unit HELD yang expired.

## Variabel Environment

| Variabel | Wajib | Deskripsi |
|---|---|---|
| `DATABASE_URL` | Ya | Connection string Postgres untuk server |
| `TEST_DATABASE_URL` | Hanya saat integration test | Connection string Postgres database test |
