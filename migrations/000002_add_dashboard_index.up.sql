-- Index composite untuk query agregasi dashboard (GROUP BY ticket_type_id, status).
-- Tanpa index ini query akan full table scan setiap refresh dashboard.
CREATE INDEX idx_ticket_units_type_status ON ticket_units (ticket_type_id, status);
