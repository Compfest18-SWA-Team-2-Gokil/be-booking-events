ALTER TABLE payments DROP COLUMN IF EXISTS xendit_refund_id;
ALTER TABLE payments DROP COLUMN IF EXISTS xendit_invoice_url;
ALTER TABLE payments RENAME COLUMN xendit_invoice_id TO external_ref;

ALTER TABLE orders DROP CONSTRAINT orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('PENDING', 'PAYMENT_PENDING', 'PAID', 'CANCELLED', 'REFUNDED'));
