# Plan: Gate Operator Assign by Username + Assignment Metadata

## Context

Sistem assign gate operator saat ini mengharuskan organizer **manual paste UUID** gate operator ke text input. Ini anti-pattern: UUID bukan identifier manusia. Selain itu, `gate_operator_assignments` tidak punya metadata (siapa assign, kapan, status), tidak ada ownership check, dan tidak ada endpoint list/revoke.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Username registration | Wajib diisi user | Natural identifier, tidak auto-generate |
| Assign by UUID vs username | Ganti total ke username | Breaking change OK karena masih dev stage |
| Revoke strategy | Soft-delete via `status=REVOKED` | Audit trail tetap ada |
| Scope | Backend only | Frontend menyusul di plan terpisah |

## Changes Overview

### Phase 1: Database Migration

**Task 1.1 — Tambah kolom `username` di tabel `users`**

File baru: `migrations/000013_add_username_to_users.up.sql`

```sql
ALTER TABLE users ADD COLUMN username TEXT;

-- Backfill: generate dari email prefix
UPDATE users SET username = split_part(email, '@', 1);

-- Resolve duplikat: append suffix angka
-- (gunakan CTE untuk deteksi duplikat dan append _1, _2, dst)
WITH duplicates AS (
    SELECT id, username,
           ROW_NUMBER() OVER (PARTITION BY username ORDER BY created_at) AS rn
    FROM users
)
UPDATE users u SET username = d.username || '_' || (d.rn - 1)::text
FROM duplicates d
WHERE u.id = d.id AND d.rn > 1;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_username_unique UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_username_format
    CHECK (username ~ '^[a-z0-9_]{3,30}$');
CREATE INDEX idx_users_username ON users (username);
```

File baru: `migrations/000013_add_username_to_users.down.sql`

```sql
DROP INDEX IF EXISTS idx_users_username;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_format;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_unique;
ALTER TABLE users DROP COLUMN IF EXISTS username;
```

**Task 1.2 — Tambah metadata di `gate_operator_assignments`**

File baru: `migrations/000014_enhance_gate_operator_assignments.up.sql`

```sql
ALTER TABLE gate_operator_assignments
    ADD COLUMN assigned_by UUID REFERENCES users(id),
    ADD COLUMN assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'REVOKED'));

-- Index untuk query active assignments
CREATE INDEX idx_gate_op_assignments_active
    ON gate_operator_assignments (event_id, user_id)
    WHERE status = 'ACTIVE';

CREATE INDEX idx_gate_op_assignments_user_active
    ON gate_operator_assignments (user_id)
    WHERE status = 'ACTIVE';
```

File baru: `migrations/000014_enhance_gate_operator_assignments.down.sql`

```sql
DROP INDEX IF EXISTS idx_gate_op_assignments_user_active;
DROP INDEX IF EXISTS idx_gate_op_assignments_active;
ALTER TABLE gate_operator_assignments
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS assigned_at,
    DROP COLUMN IF EXISTS assigned_by;
```

**Task 1.3 — Update seeds**

File: `seeds/01_users.sql` — tambah kolom `username` di INSERT:

```sql
INSERT INTO users (id, email, username, name, role, password_hash) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000001', 'organizer@compfest.id', 'organizer', 'Organizer Compfest', 'ORGANIZER', '$2a$10$...'),
  ('aaaaaaaa-0000-0000-0000-000000000002', 'buyer1@example.com', 'buyer1', 'Buyer Satu', 'BUYER', '$2a$10$...'),
  ('aaaaaaaa-0000-0000-0000-000000000003', 'buyer2@example.com', 'buyer2', 'Buyer Dua', 'BUYER', '$2a$10$...'),
  ('aaaaaaaa-0000-0000-0000-000000000004', 'gate@compfest.id', 'gate_operator', 'Gate Operator Compfest', 'GATE_OPERATOR', '$2a$10$...'),
  ('aaaaaaaa-0000-0000-0000-000000000005', 'admin@compfest.id', 'admin', 'Admin Compfest', 'ADMIN', '$2a$10$...')
ON CONFLICT (email) DO NOTHING;
```

File: `seeds/04_gate_operator_assignments.sql` — tambah `assigned_by`:

```sql
INSERT INTO gate_operator_assignments (user_id, event_id, assigned_by) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000004', 'bbbbbbbb-0000-0000-0000-000000000001', 'aaaaaaaa-0000-0000-0000-000000000001'),
  ('aaaaaaaa-0000-0000-0000-000000000004', 'bbbbbbbb-0000-0000-0000-000000000002', 'aaaaaaaa-0000-0000-0000-000000000001')
ON CONFLICT (user_id, event_id) DO NOTHING;
```

---

### Phase 2: Domain Layer

**Task 2.1 — Update `domain.User` struct**

File: `internal/auth/domain/user.go`

- Tambah field `Username string` di struct `User`
- Tambah validasi username di `Validate()`: required, regex `^[a-z0-9_]{3,30}$`
- Tambah `ErrUsernameRequired` dan `ErrInvalidUsername` di `domain/errors.go`
- Tambah `ErrUsernameAlreadyTaken` di `domain/errors.go`
- Tambah `ErrNotEventOrganizer` di `domain/errors.go` (untuk ownership check)

**Task 2.2 — Tambah domain errors baru**

File: `internal/auth/domain/errors.go`

Tambah:
```go
ErrUsernameRequired     = errors.New("username wajib diisi")
ErrInvalidUsername      = errors.New("format username tidak valid (3-30 karakter, huruf kecil, angka, underscore)")
ErrUsernameAlreadyTaken = errors.New("username sudah digunakan")
ErrNotEventOrganizer   = errors.New("bukan organizer dari event ini")
```

---

### Phase 3: Application Layer (Ports & Use Cases)

**Task 3.1 — Update `UserRepository` interface**

File: `internal/auth/application/ports.go`

Tambah method:
```go
FindByUsername(ctx context.Context, username string) (*domain.User, error)
FindByUsernameOrEmail(ctx context.Context, query string) (*domain.User, error)
AssignGateOperator(ctx context.Context, userID, eventID, assignedBy string) error
ListAssignedGateOperators(ctx context.Context, eventID string) ([]AssignedOperator, error)
RemoveGateOperator(ctx context.Context, userID, eventID string) error
SearchGateOperators(ctx context.Context, query string) ([]domain.User, error)
```

Tambah struct:
```go
type AssignedOperator struct {
    UserID     string
    Username   string
    Name       string
    Email      string
    AssignedAt time.Time
    AssignedBy string
    Status     string
}
```

**Task 3.2 — Update `AssignGateOperatorUseCase`**

File: `internal/auth/application/assign_gate_operator_usecase.go`

Perubahan:
- Input: ganti `GateOperatorUserID string` jadi `Username string`, tambah `OrganizerID string`
- Use case butuh dependency baru: `EventRepository` (dari modul events) untuk ownership check
- Alur: FindByUsername → cek role → cek organizer ownership via EventRepository.GetEvent → AssignGateOperator dengan assignedBy
- Output: return `*AssignedOperatorOutput` (bukan `error` saja), berisi info user yang di-assign
- Tambah `ErrAlreadyAssigned` jika assignment sudah ACTIVE

Constructor berubah:
```go
type AssignGateOperatorUseCase struct {
    repo      UserRepository
    eventRepo EventRepository  // cross-module dependency untuk ownership check
}
```

**Task 3.3 — Update `RegisterUseCase`**

File: `internal/auth/application/register_usecase.go`

- `RegisterInput` tambah field `Username string`
- Cek `FindByUsername` untuk duplikat sebelum create
- Set `user.Username = input.Username` sebelum `user.Validate()`

**Task 3.4 — Update `LoginUseCase`**

File: `internal/auth/application/login_usecase.go`

- Tidak ada perubahan logic. Login tetap by email + password.
- `LoginOutput.User` otomatis include `Username` karena domain struct diupdate.

**Task 3.5 — Tambah use case baru: `ListAssignedGateOperatorsUseCase`**

File baru: `internal/auth/application/list_assigned_gate_operators_usecase.go`

```go
type ListAssignedGateOperatorsUseCase struct {
    repo UserRepository
}

func (uc *ListAssignedGateOperatorsUseCase) Execute(ctx context.Context, eventID, organizerID string) ([]AssignedOperator, error) {
    // Ownership check: eventRepo.GetEvent → cek organizerID
    // Return repo.ListAssignedGateOperators(ctx, eventID)
}
```

**Task 3.6 — Tambah use case baru: `RemoveGateOperatorUseCase`**

File baru: `internal/auth/application/remove_gate_operator_usecase.go`

```go
type RemoveGateOperatorUseCase struct {
    repo      UserRepository
    eventRepo EventRepository
}

func (uc *RemoveGateOperatorUseCase) Execute(ctx context.Context, userID, eventID, organizerID string) error {
    // Ownership check
    // repo.RemoveGateOperator → UPDATE ... SET status='REVOKED' WHERE user_id=$1 AND event_id=$2 AND status='ACTIVE'
}
```

**Task 3.7 — Tambah use case baru: `SearchGateOperatorsUseCase`**

File baru: `internal/auth/application/search_gate_operators_usecase.go`

```go
type SearchGateOperatorsUseCase struct {
    repo UserRepository
}

func (uc *SearchGateOperatorsUseCase) Execute(ctx context.Context, query string) ([]domain.User, error) {
    // Return repo.SearchGateOperators(ctx, query)
    // Query: SELECT ... FROM users WHERE role='GATE_OPERATOR' AND (username ILIKE '%' || query || '%' OR name ILIKE '%' || query || '%') LIMIT 20
}
```

---

### Phase 4: Infrastructure Layer

**Task 4.1 — Update `PostgresUserRepository`**

File: `internal/auth/infrastructure/postgres_user_repository.go`

Perubahan:
- `Create`: tambah `username` di INSERT dan RETURNING
- `FindByEmail`: tambah `username` di SELECT dan Scan
- `FindByID`: tambah `username` di SELECT dan Scan
- `AssignGateOperator`: tambah parameter `assignedBy`, INSERT dengan `assigned_by` dan `status='ACTIVE'`. Gunakan `ON CONFLICT ... DO UPDATE SET status='ACTIVE', assigned_by=EXCLUDED.assigned_by, assigned_at=NOW()` agar re-assign yang di-revoke bisa di-activate kembali.
- Tambah method `FindByUsername`
- Tambah method `ListAssignedGateOperators` — JOIN users untuk dapat username/name
- Tambah method `RemoveGateOperator` — UPDATE SET status='REVOKED'
- Tambah method `SearchGateOperators` — ILIKE search pada username dan name, filter role=GATE_OPERATOR

**Task 4.2 — Update `PostgresCheckinRepository`**

File: `internal/checkin/infrastructure/postgres_checkin_repository.go`

Perubahan di `IsGateOperatorAssigned`:
```sql
SELECT EXISTS (
    SELECT 1 FROM gate_operator_assignments
    WHERE user_id = $1 AND event_id = $2
      AND status = 'ACTIVE'   -- ← tambahan
)
```

---

### Phase 5: Delivery Layer (HTTP Handlers & Routes)

**Task 5.1 — Update `AuthHandler`**

File: `internal/auth/delivery/http_handler.go`

Perubahan:
- `registerRequest`: tambah `Username string`
- `userResponse`: tambah `Username string`
- `assignGateOpRequest`: ganti `UserID` + `GateOperatorID` jadi `Username string` saja
- `AssignGateOperator` handler: ambil `organizerID` dari JWT context (`UserIDFromCtx`), pass ke use case, return info user yang di-assign (bukan hanya `{status: "assigned"}`)
- Tambah handler `ListGateOperators` — GET `/api/v1/events/{eventID}/gate-operators`
- Tambah handler `RemoveGateOperator` — DELETE `/api/v1/events/{eventID}/gate-operators/{userID}`
- Tambah handler `SearchGateOperators` — GET `/api/v1/users?role=GATE_OPERATOR&q={query}`

Response assign berubah dari:
```json
{"status": "assigned"}
```
Menjadi:
```json
{
  "status": "assigned",
  "user": {
    "id": "uuid",
    "username": "budi_sfo",
    "name": "Budi Santoso",
    "email": "budi@email.com"
  },
  "assigned_at": "2026-08-18T12:00:00Z"
}
```

**Task 5.2 — Update `AuthHandler` constructor**

File: `internal/auth/delivery/http_handler.go`

Tambah dependency:
```go
type AuthHandler struct {
    registerUC     *application.RegisterUseCase
    loginUC        *application.LoginUseCase
    assignGateOpUC *application.AssignGateOperatorUseCase
    listGateOpUC   *application.ListAssignedGateOperatorsUseCase   // baru
    removeGateOpUC *application.RemoveGateOperatorUseCase          // baru
    searchGateOpUC *application.SearchGateOperatorsUseCase         // baru
    userRepo       application.UserRepository
    redis          *redis.Client
}
```

**Task 5.3 — Update routes**

File: `cmd/server/routes/auth.go`

Tambah route:
```go
// ORGANIZER: manage gate operators
r.With(d.RequireOrganizer).Post("/api/v1/events/{eventID}/gate-operators", d.Auth.AssignGateOperator)
r.With(d.RequireOrganizer).Get("/api/v1/events/{eventID}/gate-operators", d.Auth.ListGateOperators)
r.With(d.RequireOrganizer).Delete("/api/v1/events/{eventID}/gate-operators/{userID}", d.Auth.RemoveGateOperator)

// ORGANIZER: search gate operators (for autocomplete)
r.With(d.RequireOrganizer).Get("/api/v1/users", d.Auth.SearchGateOperators)
```

**Task 5.4 — Update main.go wiring**

File: `cmd/server/main.go`

Tambah wiring untuk use case baru:
```go
// AssignGateOperatorUseCase sekarang butuh eventRepo
assignGateOpUC := authapp.NewAssignGateOperatorUseCase(userRepo, eventRepo)
listGateOpUC := authapp.NewListAssignedGateOperatorsUseCase(userRepo, eventRepo)
removeGateOpUC := authapp.NewRemoveGateOperatorUseCase(userRepo, eventRepo)
searchGateOpUC := authapp.NewSearchGateOperatorsUseCase(userRepo)
authHandler := authdelivery.NewAuthHandler(registerUC, loginUC, assignGateOpUC, listGateOpUC, removeGateOpUC, searchGateOpUC, userRepo, redisClient)
```

**Catatan penting**: `eventRepo` (dari modul events) harus di-inisialisasi **sebelum** auth use cases. Saat ini di `main.go`, events module di-wire setelah auth module. Perlu reorder atau forward-declare.

---

### Phase 6: Cross-Module Dependency

**Task 6.1 — Define `EventRepository` port di auth module**

`AssignGateOperatorUseCase` butuh cek `events.organizer_id`. Ada dua opsi:

**Opsi yang dipilih**: Definisikan interface lokal di auth/application yang hanya expose method yang dibutuhkan (anti-corruption layer):

File: `internal/auth/application/ports.go`

Tambah:
```go
type EventOwnershipChecker interface {
    GetEventOrganizerID(ctx context.Context, eventID string) (string, error)
}
```

Lalu di `main.go`, pass `eventRepo` yang sudah implement `EventRepository` (yang juga satisfy `EventOwnershipChecker` karena `GetEvent` mengembalikan `OrganizerID`).

Atau lebih sederhana: buat adapter thin di auth/infrastructure yang wrap pool pgx dan query langsung `SELECT organizer_id FROM events WHERE id = $1`.

**Rekomendasi**: Gunakan interface `EventOwnershipChecker` di auth module, dan di `main.go` buat adapter kecil:

```go
type eventOwnershipAdapter struct {
    repo eventsapp.EventRepository  // atau langsung pool
}
func (a *eventOwnershipAdapter) GetEventOrganizerID(ctx context.Context, eventID string) (string, error) {
    event, err := a.repo.GetEvent(ctx, eventID)
    if err != nil { return "", err }
    return event.OrganizerID, nil
}
```

---

### Phase 7: Tests

**Task 7.1 — Update fake repository di auth tests**

File: `internal/auth/application/fake_repository_test.go`

- Tambah `byUsername` map
- Tambah `FindByUsername` method
- Update `Create` untuk set username
- Update `AssignGateOperator` signature (tambah assignedBy)
- Tambah `ListAssignedGateOperators`, `RemoveGateOperator`, `SearchGateOperators`
- Tambah `EventOwnershipChecker` fake

**Task 7.2 — Update existing auth tests**

File: `internal/auth/application/register_usecase_test.go`
- Tambah `Username` di semua `RegisterInput`
- Tambah test case: duplicate username → `ErrUsernameAlreadyTaken`
- Tambah test case: invalid username format → `ErrInvalidUsername`
- Tambah test case: username too short → `ErrInvalidUsername`

File: `internal/auth/application/login_usecase_test.go`
- Update `seedUser` untuk include `Username`
- Update `TestAssignGateOperatorUseCase_Execute_Success` untuk input `Username` bukan `GateOperatorUserID`
- Tambah test: organizer tidak own event → `ErrNotEventOrganizer`
- Tambah test: user not found by username → `ErrUserNotFound`

**Task 7.3 — Update checkin tests**

File: `internal/checkin/application/scan_ticket_usecase_test.go`
- Tidak ada perubahan (scan flow tidak berubah, hanya query `IsGateOperatorAssigned` yang sekarang filter `status='ACTIVE'`)

File: `internal/checkin/application/fake_repository_test.go`
- Tidak ada perubahan (fake repo sudah mengabstraksi query)

**Task 7.4 — Tambah test baru untuk use case baru**

- `TestListAssignedGateOperatorsUseCase_Execute_Success`
- `TestListAssignedGateOperatorsUseCase_Execute_NotOwner`
- `TestRemoveGateOperatorUseCase_Execute_Success`
- `TestRemoveGateOperatorUseCase_Execute_NotOwner`
- `TestSearchGateOperatorsUseCase_Execute_Success`

---

## Execution Order

1. **Phase 1** — Migration files (1.1, 1.2, 1.3)
2. **Phase 2** — Domain layer (2.1, 2.2)
3. **Phase 3** — Application layer (3.1 → 3.7)
4. **Phase 6** — Cross-module dependency (6.1) — harus sebelum Phase 5
5. **Phase 4** — Infrastructure layer (4.1, 4.2)
6. **Phase 5** — Delivery layer (5.1 → 5.4)
7. **Phase 7** — Tests (7.1 → 7.4)

## Risks

| Risk | Mitigation |
|------|-----------|
| Backfill username duplikat | Migration 000013 resolve duplikat dengan suffix angka |
| Cross-module dependency auth↔events | Gunakan interface `EventOwnershipChecker` di auth, adapter di main.go |
| Breaking change: assign endpoint body berubah | Acceptable karena masih dev stage, tidak ada client production |
| Re-order main.go wiring | eventRepo harus di-init sebelum auth use cases |

## Validation

- [ ] `make test` pass setelah semua perubahan
- [ ] Migration up/down idempotent (test dengan reset DB)
- [ ] Seed data load tanpa error
- [ ] `go build ./...` tanpa compile error
- [ ] Manual test: register user baru dengan username → login → assign gate operator by username → list assigned → revoke → scan ditolak
