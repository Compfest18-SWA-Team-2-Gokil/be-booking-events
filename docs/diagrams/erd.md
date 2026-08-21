# Entity Relationship Diagram

```mermaid
erDiagram
    users {
        UUID id PK
        TEXT email UK
        TEXT username UK
        TEXT name
        TEXT role
        TEXT password_hash
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    events {
        UUID id PK
        UUID organizer_id FK
        TEXT name
        TEXT description
        TEXT category
        TEXT location
        TEXT image_url
        TIMESTAMPTZ date
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    ticket_types {
        UUID id PK
        UUID event_id FK
        TEXT name
        BIGINT price
        TEXT kind
        INT total_quota
        TEXT price_status
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    ticket_units {
        UUID id PK
        UUID ticket_type_id FK
        UUID order_id FK
        UUID admitted_by FK
        TEXT seat_label
        TEXT status
        TIMESTAMPTZ held_until
        TIMESTAMPTZ admitted_at
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    orders {
        UUID id PK
        UUID buyer_id FK
        UUID event_id FK
        TEXT status
        BIGINT total_amount
        TEXT promo_code
        BIGINT discount_amount
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    payments {
        UUID id PK
        UUID order_id FK
        BIGINT amount
        TEXT status
        TEXT payment_method
        TEXT xendit_invoice_id
        TEXT xendit_invoice_url
        TEXT xendit_refund_id
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    gate_operator_assignments {
        UUID id PK
        UUID user_id FK
        UUID event_id FK
        UUID assigned_by FK
        TEXT status
        TIMESTAMPTZ assigned_at
    }

    promos {
        UUID id PK
        UUID event_id FK
        VARCHAR code UK
        VARCHAR title
        TEXT description
        VARCHAR type
        VARCHAR discount_type
        BIGINT discount_value
        BIGINT min_order_amount
        BIGINT max_discount_amount
        INT max_usage
        INT used_count
        BOOLEAN is_active
        TIMESTAMPTZ start_date
        TIMESTAMPTZ end_date
        TIMESTAMPTZ created_at
        TIMESTAMPTZ updated_at
    }

    audit_logs {
        UUID id PK
        UUID actor_id FK
        TEXT actor_role
        TEXT entity_name
        UUID entity_id
        TEXT action
        TEXT from_state
        TEXT to_state
        JSONB metadata
        TIMESTAMPTZ created_at
    }

    users ||--o{ events : "organizer_id (organizes)"
    users ||--o{ orders : "buyer_id (places)"
    users ||--o{ gate_operator_assignments : "user_id (assigned as)"
    users ||--o{ gate_operator_assignments : "assigned_by (assigns)"
    users ||--o{ ticket_units : "admitted_by (admits)"
    users ||--o{ audit_logs : "actor_id (performs)"

    events ||--o{ ticket_types : "event_id"
    events ||--o{ orders : "event_id"
    events ||--o{ gate_operator_assignments : "event_id"
    events ||--o{ promos : "event_id"

    ticket_types ||--o{ ticket_units : "ticket_type_id"

    ticket_units }o--o| orders : "order_id"

    orders ||--o{ payments : "order_id"
```

## Keterangan Status

### `ticket_units.status`
`AVAILABLE` → `HELD` → `PAYMENT_PENDING` → `CONFIRMED` → `ADMITTED`
`HELD` / `PAYMENT_PENDING` / `CONFIRMED` → `REFUNDED`

### `orders.status`
`PENDING` → `PAYMENT_PENDING` → `PAID` → `REFUND_REQUESTED` → `REFUND_ORGANIZER_APPROVED` → `REFUNDED`
`PAYMENT_PENDING` → `PAYMENT_DISCREPANCY`
`PENDING` / `PAYMENT_PENDING` → `CANCELLED`

### `payments.status`
`PENDING` → `SUCCESS` / `FAILED` / `REFUNDED`

### `users.role`
`BUYER` | `ORGANIZER` | `GATE_OPERATOR` | `ADMIN`

### `ticket_types.kind`
`GA` (General Admission) | `SEATED`

### `promos.type`
`VOUCHER` (global) | `PROMO` (spesifik per event)

### `promos.discount_type`
`PERCENTAGE` | `FIXED_AMOUNT`
