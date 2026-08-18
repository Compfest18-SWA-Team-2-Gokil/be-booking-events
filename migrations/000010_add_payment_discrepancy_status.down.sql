DROP INDEX IF EXISTS idx_orders_disputes;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders
    ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING', 'PAYMENT_PENDING', 'PAID', 'CANCELLED', 'REFUNDED'));
