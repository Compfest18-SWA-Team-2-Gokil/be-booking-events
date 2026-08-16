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
