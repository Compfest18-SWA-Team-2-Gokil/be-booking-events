-- seeds/01_users.sql
-- Seed user awal untuk development & testing
-- Password semua adalah: "Password123!" (bcrypt hash)

INSERT INTO users (id, email, username, name, role, password_hash) VALUES
  (
    'aaaaaaaa-0000-0000-0000-000000000001',
    'organizer@compfest.id',
    'organizer',
    'Organizer Compfest',
    'ORGANIZER',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000002',
    'buyer1@example.com',
    'buyer1',
    'Buyer Satu',
    'BUYER',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000003',
    'buyer2@example.com',
    'buyer2',
    'Buyer Dua',
    'BUYER',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000004',
    'gate@compfest.id',
    'gate_operator',
    'Gate Operator Compfest',
    'GATE_OPERATOR',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000005',
    'admin@compfest.id',
    'admin',
    'Admin Compfest',
    'ADMIN',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  )
ON CONFLICT (email) DO NOTHING;
