-- TEXT + CHECK constraint (bukan ENUM) untuk kemudahan scanning di pgx v5 tanpa registrasi tipe.

CREATE TABLE events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    date        TIMESTAMPTZ NOT NULL,
    location    TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ticket_types (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID NOT NULL REFERENCES events(id),
    name         TEXT NOT NULL,
    price        BIGINT NOT NULL CHECK (price >= 0),
    kind         TEXT NOT NULL CHECK (kind IN ('GA', 'SEATED')),
    total_quota  INT NOT NULL CHECK (total_quota > 0),
    -- OPEN: harga masih bisa diubah Organizer. LOCKED: final, tidak berubah walau tiket resale.
    price_status TEXT NOT NULL DEFAULT 'OPEN' CHECK (price_status IN ('OPEN', 'LOCKED')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ticket_units (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_type_id  UUID NOT NULL REFERENCES ticket_types(id),
    seat_label      TEXT,           -- NULL untuk GA, diisi untuk Seated (e.g. "Blok B - Baris 1 - Kursi 4")
    status          TEXT NOT NULL DEFAULT 'AVAILABLE'
                        CHECK (status IN ('AVAILABLE', 'HELD', 'PAYMENT_PENDING', 'CONFIRMED', 'REFUNDED', 'ADMITTED')),
    held_until      TIMESTAMPTZ,    -- diisi saat HELD, NULL di status lain
    order_id        UUID,           -- FK ke tabel orders (diisi modul PRD-04 nanti)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hot path anti-oversell:
--   SELECT ... WHERE ticket_type_id = X AND status = 'AVAILABLE' ORDER BY id LIMIT N FOR UPDATE
-- Partial index hanya cover baris AVAILABLE → lebih kecil dan lebih cepat di-scan.
CREATE INDEX idx_ticket_units_available
    ON ticket_units (ticket_type_id, id)
    WHERE status = 'AVAILABLE';

-- Cleanup worker:
--   UPDATE ... WHERE status = 'HELD' AND held_until < NOW() FOR UPDATE SKIP LOCKED
CREATE INDEX idx_ticket_units_held_expired
    ON ticket_units (held_until)
    WHERE status = 'HELD';
