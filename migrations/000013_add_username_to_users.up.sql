ALTER TABLE users ADD COLUMN username TEXT;

UPDATE users SET username = split_part(email, '@', 1);

WITH duplicates AS (
    SELECT id, username,
           ROW_NUMBER() OVER (PARTITION BY username ORDER BY created_at) AS rn
    FROM users
)
UPDATE users u SET username = d.username || '_' || (d.rn - 1)::text
FROM duplicates d
WHERE u.id = d.id AND d.rn > 1;

ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_username_unique UNIQUE (username);
ALTER TABLE users ADD CONSTRAINT users_username_format
    CHECK (username ~ '^[a-z0-9_]{3,30}$');
CREATE INDEX idx_users_username ON users (username);
