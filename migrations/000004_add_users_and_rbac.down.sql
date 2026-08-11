ALTER TABLE ticket_units DROP CONSTRAINT IF EXISTS fk_ticket_units_admitted_by;
ALTER TABLE ticket_units ALTER COLUMN admitted_by TYPE TEXT USING admitted_by::TEXT;
ALTER TABLE events DROP COLUMN IF EXISTS organizer_id;
DROP TABLE IF EXISTS gate_operator_assignments;
DROP TABLE IF EXISTS users;
