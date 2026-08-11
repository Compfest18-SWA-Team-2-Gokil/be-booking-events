# Entity-Relationship Diagram

> Diagram di bawah dirender otomatis oleh GitHub (Mermaid).  
> Tabel dengan label ✅ sudah ada. Tabel dengan label 🔲 masih perlu diimplementasikan.

```mermaid
erDiagram

    %% ─── EXISTING TABLES ────────────────────────────────────────────────────

    events["events ✅"] {
        uuid        id              PK
        uuid        organizer_id    FK "→ users.id 🔲"
        text        name
        timestamptz date
        text        location
        timestamptz created_at
        timestamptz updated_at
    }

    ticket_types["ticket_types ✅"] {
        uuid        id           PK
        uuid        event_id     FK
        text        name
        bigint      price
        text        kind         "GA | SEATED"
        int         total_quota
        text        price_status "OPEN | LOCKED"
        timestamptz created_at
        timestamptz updated_at
    }

    ticket_units["ticket_units ✅"] {
        uuid        id              PK
        uuid        ticket_type_id  FK
        text        seat_label      "NULL untuk GA"
        text        status          "AVAILABLE | HELD | PAYMENT_PENDING | CONFIRMED | ADMITTED | REFUNDED"
        timestamptz held_until
        uuid        order_id        FK "→ orders.id 🔲"
        timestamptz admitted_at
        uuid        admitted_by     FK "→ users.id 🔲"
        timestamptz created_at
        timestamptz updated_at
    }

    %% ─── NEW TABLES (PERLU DIIMPLEMENTASIKAN) ───────────────────────────────

    users["users 🔲"] {
        uuid        id            PK
        text        email         UK
        text        name
        text        role          "BUYER | ORGANIZER | GATE_OPERATOR | ADMIN"
        text        password_hash
        timestamptz created_at
        timestamptz updated_at
    }

    gate_operator_assignments["gate_operator_assignments 🔲"] {
        uuid id       PK
        uuid user_id  FK "→ users.id"
        uuid event_id FK "→ events.id"
    }

    orders["orders 🔲"] {
        uuid        id           PK
        uuid        buyer_id     FK "→ users.id"
        uuid        event_id     FK "→ events.id"
        text        status       "PENDING | PAYMENT_PENDING | PAID | CANCELLED | REFUNDED"
        bigint      total_amount
        timestamptz created_at
        timestamptz updated_at
    }

    payments["payments 🔲"] {
        uuid        id             PK
        uuid        order_id       FK "→ orders.id"
        bigint      amount
        text        status         "PENDING | SUCCESS | FAILED | REFUNDED"
        text        payment_method
        text        external_ref   "reference dari payment gateway"
        timestamptz created_at
        timestamptz updated_at
    }

    %% ─── RELASI ─────────────────────────────────────────────────────────────

    users                    ||--o{ events                      : "organizer_id (belum ada FK)"
    events                   ||--o{ ticket_types                : "event_id"
    ticket_types             ||--o{ ticket_units                : "ticket_type_id"
    users                    ||--o{ orders                      : "buyer_id"
    events                   ||--o{ orders                      : "event_id"
    orders                   ||--o{ ticket_units                : "order_id (belum ada FK)"
    orders                   ||--o{ payments                    : "order_id"
    users                    ||--o{ gate_operator_assignments   : "user_id"
    events                   ||--o{ gate_operator_assignments   : "event_id"
    users                    ||--o{ ticket_units                : "admitted_by (belum ada FK)"
```

---

## Status Tabel

| Tabel | Status | Modul | Migration |
|---|---|---|---|
| `events` | ✅ Ada | Inventory (PRD-01) | `000001` |
| `ticket_types` | ✅ Ada | Inventory (PRD-01) | `000001` |
| `ticket_units` | ✅ Ada + extended | Inventory + Check-in | `000001`, `000003` |
| `users` | 🔲 Belum ada | Auth (TDD-02) | `000004` |
| `gate_operator_assignments` | 🔲 Belum ada | Auth/RBAC (TDD-02) | `000004` |
| `orders` | 🔲 Belum ada | Orders (PRD-04) | `000005` |
| `payments` | 🔲 Belum ada | Payments | `000005` |

---

## Catatan Desain

### Kolom yang perlu diupgrade menjadi FK proper

Kolom-kolom berikut saat ini bertipe `UUID` atau `TEXT` tanpa FK constraint — akan di-enforce setelah tabel target dibuat:

| Kolom | Tabel | Target FK |
|---|---|---|
| `ticket_units.order_id` | ticket_units | `orders.id` |
| `ticket_units.admitted_by` | ticket_units | `users.id` |
| `events.organizer_id` | events | `users.id` (perlu ditambah kolom) |

### Status Machine `orders`

```
PENDING → PAYMENT_PENDING → PAID
                          ↘
                        CANCELLED
PAID → REFUNDED
```

Sinkron dengan `ticket_units.status`: saat order PAID → semua ticket_units-nya CONFIRMED. Saat order REFUNDED → semua ticket_units-nya REFUNDED.

### RBAC Gate Operator

`gate_operator_assignments` adalah junction table many-to-many antara `users` (role = GATE_OPERATOR) dan `events`. Gate operator hanya bisa scan tiket dari event yang ada di tabel ini — di-enforce di middleware sebelum `ScanTicketUseCase`.
