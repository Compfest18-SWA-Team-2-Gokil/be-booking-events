-- Tambah status REFUND_REQUESTED ke orders.
ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING', 'PAYMENT_PENDING', 'PAID', 'CANCELLED', 'REFUND_REQUESTED', 'REFUNDED'));

-- Ganti external_ref dengan kolom spesifik Xendit.
ALTER TABLE payments RENAME COLUMN external_ref TO xendit_invoice_id;
ALTER TABLE payments ADD COLUMN xendit_invoice_url TEXT;
ALTER TABLE payments ADD COLUMN xendit_refund_id   TEXT;
