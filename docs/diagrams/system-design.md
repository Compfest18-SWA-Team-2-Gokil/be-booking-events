# System Design Diagram

## Arsitektur Deployment

```mermaid
graph TD
    CLIENT["🌐 Client\n(Browser / Mobile)"]

    subgraph SERVER ["☁️ VPS — compfest-team2"]
        NGINX["🔀 Nginx\n:443 HTTPS\n:80 → redirect HTTPS"]
        APP["⚙️ Go Backend\nlocalhost:8080\nservice: ticketing-app"]
        MINIO["🗄️ MinIO\nlocalhost:9002 API\nlocalhost:9001 Console\nservice: minio"]
    end

    subgraph EXTERNAL ["☁️ Eksternal Services"]
        PG[("🐘 PostgreSQL\nRailway")]
        REDIS[("⚡ Redis\nRedis Cloud")]
        XENDIT["💳 Xendit\nPayment Gateway"]
    end

    CLIENT -->|"HTTPS :443"| NGINX
    NGINX -->|"proxy_pass /api/v1/* /docs\nlocalhost:8080"| APP
    NGINX -->|"proxy_pass /event-images/*\nlocalhost:9002"| MINIO
    APP -->|"pgx/v5\nDATABASE_URL"| PG
    APP -->|"go-redis\nREDIS_URL"| REDIS
    APP -->|"REST API\nXENDIT_SECRET_KEY"| XENDIT
    XENDIT -->|"Webhook POST /api/v1/orders/webhook/xendit"| APP
    APP -->|"minio-go/v7\nMINIO_ENDPOINT"| MINIO
```

## Arsitektur Aplikasi (Clean Architecture)

```mermaid
graph TD
    subgraph DELIVERY ["Delivery Layer (HTTP Handlers)"]
        H1["auth/delivery"]
        H2["events/delivery"]
        H3["orders/delivery"]
        H4["checkin/delivery"]
        H5["queue/delivery"]
        H6["inventory/delivery"]
        H7["dashboard/delivery"]
        H8["promos/delivery"]
        H9["admin/delivery"]
    end

    subgraph APPLICATION ["Application Layer (Use Cases)"]
        A1["RegisterUC\nLoginUC"]
        A2["CreateEventUC\nUploadImageUC"]
        A3["CreateOrderUC\nConfirmPaymentUC\nRequestRefundUC\nApproveRefundUC"]
        A4["IssueTicketUC\nScanTicketUC"]
        A5["JoinQueueUC\nValidateTokenUC"]
        A6["HoldTicketUC\nExpireHeldUC"]
        A7["GetMetricsUC"]
        A8["ValidatePromoUC\nAdminPromoUC"]
        A9["ReassignTicketUC"]
    end

    subgraph DOMAIN ["Domain Layer (Entities & Rules)"]
        D1["User\nRole"]
        D2["Event"]
        D3["Order\nOrderStatus"]
        D4["TicketQR"]
        D5["QueueToken"]
        D6["TicketUnit\nTicketUnitStatus"]
        D7["Promo"]
    end

    subgraph INFRA ["Infrastructure Layer (DB & External)"]
        I1["postgres_auth_repository"]
        I2["postgres_event_repository\nminio_storage_provider"]
        I3["postgres_order_repository\nxendit_payment_provider"]
        I4["postgres_checkin_repository"]
        I5["redis_queue_repository"]
        I6["postgres_ticket_repository\ncron_expiry_worker"]
        I7["postgres_promo_repository"]
        I8["postgres_admin_repository"]
    end

    H1 --> A1 --> D1
    H2 --> A2 --> D2
    H3 --> A3 --> D3
    H4 --> A4 --> D4
    H5 --> A5 --> D5
    H6 --> A6 --> D6
    H8 --> A8 --> D7
    H9 --> A9

    A1 --> I1
    A2 --> I2
    A3 --> I3
    A6 --> I6
    A4 --> I4
    A5 --> I5
    A7 --> I6
    A8 --> I7
    A9 --> I8
```

## Alur Anti-Oversell (Hot Path)

```mermaid
sequenceDiagram
    participant B as Buyer
    participant Q as Queue Guard
    participant H as HoldTicket Handler
    participant DB as PostgreSQL

    B->>Q: POST /api/v1/tickets/hold
    Q-->>B: 429 (jika traffic > threshold)

    B->>H: POST /api/v1/tickets/hold
    H->>DB: BEGIN TRANSACTION
    H->>DB: UPDATE HELD expired → AVAILABLE (lazy expiry)
    H->>DB: SELECT FOR UPDATE ORDER BY id ASC LIMIT N
    alt stok tersedia
        H->>DB: UPDATE status = HELD, held_until = NOW()+5m
        H->>DB: COMMIT
        H-->>B: 200 OK (unit IDs)
    else stok habis
        H->>DB: ROLLBACK
        H-->>B: 409 ErrTicketNotAvailable
    end
```

## Alur Pembayaran

```mermaid
sequenceDiagram
    participant B as Buyer
    participant API as Go Backend
    participant X as Xendit
    participant DB as PostgreSQL

    B->>API: POST /api/v1/orders (hold_ids)
    API->>DB: CREATE order (PENDING)
    API->>X: POST /v2/invoices
    X-->>API: invoice_url, invoice_id
    API->>DB: CREATE payment (PENDING), UPDATE order (PAYMENT_PENDING)
    API-->>B: {order_id, invoice_url}

    B->>X: Bayar via invoice
    X->>API: Webhook POST /orders/webhook/xendit
    API->>DB: UPDATE payment (SUCCESS), order (PAID)
    API->>DB: UPDATE ticket_units → CONFIRMED
    API-->>X: 200 OK
```

## Alur Queue / Waiting Room

```mermaid
sequenceDiagram
    participant B as Buyer
    participant API as Go Backend
    participant R as Redis

    B->>API: POST /events/{id}/queue/join
    API->>R: RPUSH queue:{eventID} userID
    API-->>B: {position, estimated_wait}

    loop setiap 5 detik
        API->>R: Release 10 user dari queue
        API->>R: SET token:{userID} (TTL 10 menit)
    end

    B->>API: GET /events/{id}/queue/status
    API->>R: Cek posisi / token
    API-->>B: {status: "ready", token: "..."}

    B->>API: POST /tickets/hold (X-Queue-Token header)
    API->>R: Validate token
    API-->>B: ticket units held
```

## Infrastruktur CI/CD

```mermaid
graph LR
    DEV["👨‍💻 Developer"]
    GH["GitHub\nmain branch"]

    subgraph CI ["CI — ubuntu-latest"]
        CI1["go build ./..."]
        CI2["go vet ./..."]
        CI3["Unit Tests\n(domain + application)"]
        CI4["Integration Tests\n(infrastructure + Postgres)"]
    end

    subgraph CD ["CD — ubuntu-latest"]
        CD1["Build binary\nlinux/amd64"]
        CD2["Stop service\n(systemctl stop)"]
        CD3["SCP binary +\nmigrations ke server"]
        CD4["chmod +x\nsystemctl start"]
        CD5["Health check\n/docs retry 10x"]
    end

    SERVER["🖥️ VPS\nauto-migration\nsaat startup"]

    DEV -->|"git push"| GH
    GH --> CI
    CI -->|"merge ke main"| CD
    CD1 --> CD2 --> CD3 --> CD4 --> CD5
    CD5 -->|"deploy OK"| SERVER
```
