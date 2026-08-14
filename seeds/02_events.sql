-- seeds/02_events.sql
-- Seed event sample untuk development

INSERT INTO events (id, name, date, location, organizer_id) VALUES
  (
    'bbbbbbbb-0000-0000-0000-000000000001',
    'Pasar Senen Tour 2027',
    '2027-06-15 12:00:00+07',
    'Pasarsenen, Jakarta',
    'aaaaaaaa-0000-0000-0000-000000000001'
  ),
  (
    'bbbbbbbb-0000-0000-0000-000000000002',
    'Compfest Music Festival',
    '2027-09-20 16:00:00+07',
    'Balairung UI, Depok',
    'aaaaaaaa-0000-0000-0000-000000000001'
  ),
  (
    'bbbbbbbb-0000-0000-0000-000000000003',
    'Tech Conference Jakarta',
    '2027-11-05 09:00:00+07',
    'JCC Senayan, Jakarta',
    'aaaaaaaa-0000-0000-0000-000000000001'
  )
ON CONFLICT DO NOTHING;
