CREATE TABLE IF NOT EXISTS promos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL UNIQUE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    event_id UUID REFERENCES events(id) ON DELETE SET NULL,
    discount_type VARCHAR(20) NOT NULL, -- 'PERCENTAGE' atau 'FIXED_AMOUNT'
    discount_value BIGINT NOT NULL,      -- e.g. 20 (untuk 20%) atau 50000 (untuk Rp 50.000)
    min_order_amount BIGINT DEFAULT 0,
    max_discount_amount BIGINT DEFAULT 0, -- 0 jika tanpa limit
    max_usage INT DEFAULT 0,             -- 0 jika unlimited
    used_count INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_promos_code ON promos(code);
CREATE INDEX IF NOT EXISTS idx_promos_event_id ON promos(event_id);

-- Tambahkan kolom promo_code dan discount_amount pada tabel orders
ALTER TABLE orders 
ADD COLUMN IF NOT EXISTS promo_code VARCHAR(50),
ADD COLUMN IF NOT EXISTS discount_amount BIGINT DEFAULT 0;
