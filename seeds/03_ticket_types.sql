-- seeds/03_ticket_types.sql
-- Seed ticket type untuk setiap event

INSERT INTO ticket_types (id, event_id, name, price, kind, total_quota, price_status) VALUES
  -- Pasar Senen Tour
  (
    'cccccccc-0000-0000-0000-000000000001',
    'bbbbbbbb-0000-0000-0000-000000000001',
    'Reguler',
    150000,
    'GA',
    500,
    'OPEN'
  ),
  (
    'cccccccc-0000-0000-0000-000000000002',
    'bbbbbbbb-0000-0000-0000-000000000001',
    'VIP',
    350000,
    'GA',
    100,
    'OPEN'
  ),
  -- Compfest Music Festival
  (
    'cccccccc-0000-0000-0000-000000000003',
    'bbbbbbbb-0000-0000-0000-000000000002',
    'Early Bird',
    99000,
    'GA',
    200,
    'LOCKED'
  ),
  (
    'cccccccc-0000-0000-0000-000000000004',
    'bbbbbbbb-0000-0000-0000-000000000002',
    'Regular GA',
    150000,
    'GA',
    1000,
    'OPEN'
  ),
  (
    'cccccccc-0000-0000-0000-000000000005',
    'bbbbbbbb-0000-0000-0000-000000000002',
    'Seated Blok A',
    500000,
    'SEATED',
    50,
    'OPEN'
  ),
  -- Tech Conference
  (
    'cccccccc-0000-0000-0000-000000000006',
    'bbbbbbbb-0000-0000-0000-000000000003',
    'General Admission',
    0,
    'GA',
    300,
    'LOCKED'
  )
ON CONFLICT DO NOTHING;
