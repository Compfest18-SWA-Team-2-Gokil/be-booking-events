-- Tambah status REFUND_ORGANIZER_APPROVED ke constraint orders untuk alur 2-tahap (Organizer -> Admin).
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_status_check;

ALTER TABLE orders
    ADD CONSTRAINT orders_status_check
    CHECK (status IN (
        'PENDING',
        'PAYMENT_PENDING',
        'PAID',
        'CANCELLED',
        'REFUNDED',
        'REFUND_REQUESTED',
        'REFUND_ORGANIZER_APPROVED',
        'PAYMENT_DISCREPANCY'
    ));

CREATE INDEX IF NOT EXISTS idx_orders_disputes
    ON orders (status)
    WHERE status IN ('PAYMENT_DISCREPANCY', 'REFUND_REQUESTED', 'REFUND_ORGANIZER_APPROVED');
