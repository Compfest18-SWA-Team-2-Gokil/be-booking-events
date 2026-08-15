ALTER TABLE events
    ADD COLUMN description TEXT NOT NULL DEFAULT '',
    ADD COLUMN category    TEXT NOT NULL DEFAULT 'music'
        CHECK (category IN ('music', 'olahraga', 'seni', 'workshop'));

CREATE INDEX idx_events_category ON events (category);
