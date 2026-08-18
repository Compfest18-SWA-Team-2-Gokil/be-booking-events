DROP INDEX IF EXISTS idx_gate_op_assignments_user_active;
DROP INDEX IF EXISTS idx_gate_op_assignments_active;
ALTER TABLE gate_operator_assignments
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS assigned_at,
    DROP COLUMN IF EXISTS assigned_by;
