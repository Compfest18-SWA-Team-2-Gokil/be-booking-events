-- Kolom tambahan untuk modul check-in (PRD-06).
-- admitted_at: timestamp saat tiket di-scan masuk.
-- admitted_by: identitas gate operator yang melakukan scan.
ALTER TABLE ticket_units ADD COLUMN admitted_at TIMESTAMPTZ;
ALTER TABLE ticket_units ADD COLUMN admitted_by TEXT;
