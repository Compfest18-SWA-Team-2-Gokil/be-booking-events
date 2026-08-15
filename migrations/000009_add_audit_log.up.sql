CREATE TABLE audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID        REFERENCES users(id),
    actor_role  TEXT        NOT NULL,
    entity_type TEXT        NOT NULL,  -- 'order' | 'ticket_unit' | 'payment'
    entity_id   UUID        NOT NULL,
    action      TEXT        NOT NULL,  -- 'CONFIRM_PAYMENT' | 'LOST_SEAT' | 'REFUND_REQUESTED' | 'ADMITTED' | 'ADMIN_OVERRIDE'
    from_status TEXT,
    to_status   TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Append-only: tidak ada UPDATE/DELETE, hanya INSERT + SELECT.
-- Index untuk query dispute dashboard dan audit trail per-entity.
CREATE INDEX idx_audit_logs_entity    ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_logs_action    ON audit_logs (action);
CREATE INDEX idx_audit_logs_created   ON audit_logs (created_at DESC);

-- Revoke UPDATE dan DELETE dari app user agar benar-benar immutable.
-- (Jalankan sebagai superuser jika ingin enforce di DB level.)
-- REVOKE UPDATE, DELETE ON audit_logs FROM booking_app_user;
