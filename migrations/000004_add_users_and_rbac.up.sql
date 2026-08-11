-- Modul Auth (TDD-02).
-- users adalah single table untuk semua role; role menentukan akses di application layer.

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('BUYER', 'ORGANIZER', 'GATE_OPERATOR', 'ADMIN')),
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- RBAC: gate operator hanya bisa scan event yang ia di-assign.
CREATE TABLE gate_operator_assignments (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    event_id  UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    UNIQUE (user_id, event_id)
);

-- Tambah organizer_id ke events (nullable untuk backward-compat data existing).
ALTER TABLE events ADD COLUMN organizer_id UUID REFERENCES users(id);

-- Kolom admitted_by di ticket_units: ubah dari TEXT ke UUID (soft-ref ke users).
-- Data existing (dev/test) bisa diabaikan karena admitted_by masih NULL.
ALTER TABLE ticket_units ALTER COLUMN admitted_by TYPE UUID USING admitted_by::UUID;
ALTER TABLE ticket_units ADD CONSTRAINT fk_ticket_units_admitted_by
    FOREIGN KEY (admitted_by) REFERENCES users(id);

CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_gate_operator_assignments_event ON gate_operator_assignments (event_id);
