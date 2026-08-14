DROP INDEX IF EXISTS idx_events_category;

ALTER TABLE events
    DROP COLUMN IF EXISTS description,
    DROP COLUMN IF EXISTS category;
