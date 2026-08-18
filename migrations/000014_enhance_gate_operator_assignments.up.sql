ALTER TABLE gate_operator_assignments
    ADD COLUMN assigned_by UUID REFERENCES users(id),
    ADD COLUMN assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'REVOKED'));

CREATE INDEX idx_gate_op_assignments_active
    ON gate_operator_assignments (event_id, user_id)
    WHERE status = 'ACTIVE';

CREATE INDEX idx_gate_op_assignments_user_active
    ON gate_operator_assignments (user_id)
    WHERE status = 'ACTIVE';
