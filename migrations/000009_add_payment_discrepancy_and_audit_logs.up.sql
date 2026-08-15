-- Tambah status PAYMENT_DISCREPANCY ke orders untuk penanganan Lost Seat edge-case (PRD-05, PRD-10)
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING', 'PAYMENT_PENDING', 'PAID', 'CANCELLED', 'REFUND_REQUESTED', 'REFUNDED', 'PAYMENT_DISCREPANCY'));

-- Buat tabel audit_logs untuk traceability & immutable audit trail (PRD-11)
CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID,
    actor_role  TEXT,
    entity_name TEXT NOT NULL,
    entity_id   UUID NOT NULL,
    action      TEXT NOT NULL,
    from_state  TEXT,
    to_state    TEXT,
    metadata    JSONB DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_name, entity_id);
CREATE INDEX idx_audit_logs_created ON audit_logs (created_at DESC);
