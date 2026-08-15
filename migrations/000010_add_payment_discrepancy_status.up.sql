-- Tambah status PAYMENT_DISCREPANCY dan REFUND_REQUESTED ke constraint orders.
-- Hapus constraint lama, buat yang baru dengan nilai lebih lengkap.
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
        'PAYMENT_DISCREPANCY'
    ));

-- Index tambahan untuk dispute dashboard (query by anomali status).
CREATE INDEX IF NOT EXISTS idx_orders_disputes
    ON orders (status)
    WHERE status IN ('PAYMENT_DISCREPANCY', 'REFUND_REQUESTED');
