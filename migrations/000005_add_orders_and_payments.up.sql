-- Modul Orders + Payments (PRD-04).

CREATE TABLE orders (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    buyer_id     UUID NOT NULL REFERENCES users(id),
    event_id     UUID NOT NULL REFERENCES events(id),
    status       TEXT NOT NULL DEFAULT 'PENDING'
                     CHECK (status IN ('PENDING', 'PAYMENT_PENDING', 'PAID', 'CANCELLED', 'REFUNDED')),
    total_amount BIGINT NOT NULL CHECK (total_amount >= 0),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE payments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       UUID NOT NULL REFERENCES orders(id),
    amount         BIGINT NOT NULL CHECK (amount > 0),
    status         TEXT NOT NULL DEFAULT 'PENDING'
                       CHECK (status IN ('PENDING', 'SUCCESS', 'FAILED', 'REFUNDED')),
    payment_method TEXT,
    external_ref   TEXT,   -- reference ID dari payment gateway (Midtrans, dll)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enforce FK yang sebelumnya soft: ticket_units.order_id → orders.id
ALTER TABLE ticket_units ADD CONSTRAINT fk_ticket_units_order
    FOREIGN KEY (order_id) REFERENCES orders(id);

CREATE INDEX idx_orders_buyer ON orders (buyer_id);
CREATE INDEX idx_orders_event ON orders (event_id);
CREATE INDEX idx_orders_status ON orders (status) WHERE status NOT IN ('PAID', 'CANCELLED', 'REFUNDED');
CREATE INDEX idx_payments_order ON payments (order_id);
