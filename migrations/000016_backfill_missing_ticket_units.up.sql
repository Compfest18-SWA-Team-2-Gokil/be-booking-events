INSERT INTO ticket_units (ticket_type_id, status)
SELECT tt.id, 'AVAILABLE'
FROM ticket_types tt
CROSS JOIN LATERAL generate_series(1, tt.total_quota)
WHERE NOT EXISTS (
    SELECT 1 FROM ticket_units tu WHERE tu.ticket_type_id = tt.id
);
