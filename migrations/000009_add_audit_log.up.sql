CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID        REFERENCES users(id),
    actor_role  TEXT        NOT NULL,
    entity_name TEXT        NOT NULL,
    entity_id   UUID        NOT NULL,
    action      TEXT        NOT NULL,
    from_state  TEXT,
    to_state    TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_entity    ON audit_logs (entity_name, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action    ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created   ON audit_logs (created_at DESC);
