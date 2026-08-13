-- seeds/04_gate_operator_assignments.sql
-- Assign gate operator ke event

INSERT INTO gate_operator_assignments (user_id, event_id) VALUES
  ('aaaaaaaa-0000-0000-0000-000000000004', 'bbbbbbbb-0000-0000-0000-000000000001'),
  ('aaaaaaaa-0000-0000-0000-000000000004', 'bbbbbbbb-0000-0000-0000-000000000002')
ON CONFLICT (user_id, event_id) DO NOTHING;
