-- seeds/01_users.sql
-- Seed user awal untuk development & testing
-- Password semua adalah: "Password123!" (bcrypt hash)

INSERT INTO users (id, email, name, role, password_hash) VALUES
  (
    'aaaaaaaa-0000-0000-0000-000000000001',
    'organizer@compfest.id',
    'Organizer Compfest',
    'ORGANIZER',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'  -- password: Password123!
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000002',
    'buyer1@example.com',
    'Buyer Satu',
    'BUYER',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000003',
    'buyer2@example.com',
    'Buyer Dua',
    'BUYER',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  ),
  (
    'aaaaaaaa-0000-0000-0000-000000000004',
    'gate@compfest.id',
    'Gate Operator Compfest',
    'GATE_OPERATOR',
    '$2a$10$1JH5QPVXuPlnwr565gMeReXFnfUJjDb/oDaylX4bQnbXuS3JxWjPy'
  )
ON CONFLICT (email) DO NOTHING;
