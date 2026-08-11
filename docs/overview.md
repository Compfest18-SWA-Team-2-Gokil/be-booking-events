# Dokumentasi Analisis Aplikasi Booking Events

## Ringkasan Aplikasi
Aplikasi ini adalah backend service untuk sistem ticketing event (pemesanan tiket) yang ditulis dengan Golang. Arsitektur yang digunakan adalah **Clean Architecture** dengan pendekatan Domain-Driven Design (DDD) secara struktural, terbagi menjadi layer domain, application, delivery, dan infrastructure. Aplikasi ini memiliki fitur utama manajemen inventori tiket yang menggunakan mekanisme *pessimistic locking* (`SELECT FOR UPDATE SKIP LOCKED`) di PostgreSQL untuk mencegah overbooking (oversell).

## Fitur yang Sudah Ada (Implemented)
- **Hold Ticket (Pemesanan Sementara)**: Endpoint `POST /api/v1/tickets/hold` yang memungkinkan pemesanan tiket dengan status "HELD" sementara waktu (5 menit).
- **Anti-Oversell Engine**: Repositori Postgres yang mengimplementasikan konkurensi aman untuk pemesanan tiket menggunakan query `SELECT FOR UPDATE SKIP LOCKED`. Transaksi bersifat *all-or-nothing*.
- **Cron Worker Expiry**: Pekerja latar belakang (background worker) yang berjalan setiap 30 detik untuk mereset tiket yang lewat masa *hold*-nya (expired) kembali menjadi `AVAILABLE`.
- **Domain Logic**: Model dasar untuk `Event`, `TicketType`, dan `TicketUnit` beserta *rules* validasi dan perubahan state sederhana.
- **Unit & Integration Tests**: Pengujian mencakup pengujian memori statis (fake repository) dan pengujian integrasi database real yang disimulasikan menggunakan testcontainers atau docker database lokal dengan 50 goroutine (concurrent testing).

## Fitur yang Belum Dilakukan (To Be Implemented)
- **Sistem Pembayaran / Pemesanan Akhir (Checkout)**: Saat ini tiket baru bisa di-*hold*. Belum ada fungsionalitas untuk mengkonfirmasi tiket ke `PAYMENT_PENDING` lalu ke `CONFIRMED`.
- **Manajemen Event & Tipe Tiket**: Endpoint atau fitur CRUD untuk membuat, membaca, mengubah, atau menghapus Data `Event` dan `TicketType` oleh Organizer.
- **Cancel/Refund Tiket**: Transisi tiket ke status `REFUNDED`.
- **Auto Migration Runner**: Migrasi database (`.sql`) masih harus dijalankan secara manual via *command line*. Idealnya ada tool seperti `golang-migrate` yang ditanamkan langsung saat aplikasi *start*.
- **Authentication & Authorization**: Belum ada pengamanan API menggunakan JWT atau sesi login.
- **Manajemen Pemesanan (Orders)**: Di skema database terdapat kolom `order_id` tapi modul ini belum dibuat sama sekali.

## Struktur Folder dan File

| Folder / File | Deskripsi / Peran |
| ------------- | ----------------- |
| **`cmd/`** | Entrypoint aplikasi. |
| └ `cmd/server/main.go` | Inisialisasi dependensi, koneksi database, router HTTP, Cron Worker, dan menjalankan web server. |
| **`docs/`** | Dokumentasi proyek. |
| **`internal/`** | Kode privat aplikasi. |
| └ **`internal/inventory/`** | Modul domain untuk inventori tiket. |
| &nbsp;&nbsp;&nbsp; └ **`domain/`** | Entitas utama bisnis tanpa dependensi luar. Terdapat `event.go`, `ticket_type.go`, `ticket_unit.go`, dan *custom errors*. |
| &nbsp;&nbsp;&nbsp; └ **`application/`** | Orchestrator logika aplikasi (Use Case). Berisi `hold_ticket_usecase.go`, `expire_held_tickets_usecase.go`, dan `ports.go` (interface repository). |
| &nbsp;&nbsp;&nbsp; └ **`delivery/`** | Interface dengan dunia luar (HTTP). Berisi `http_handler.go` untuk menangani request dan response JSON endpoint `chi`. |
| &nbsp;&nbsp;&nbsp; └ **`infrastructure/`** | Implementasi layer eksternal. Berisi `postgres_ticket_repository.go` (interaksi pgx dengan postgres) dan `cron_expiry_worker.go` (worker penjadwalan). |
| **`migrations/`** | Berisi file SQL untuk migrasi database. |
| └ `000001_create_inventory_tables.up.sql` | Skema pembuatan tabel `events`, `ticket_types`, `ticket_units` dan index untuk tabel. |
| └ `000001_create_inventory_tables.down.sql`| Script untuk menghapus tabel yang dibuat di versi ini. |
| **`.env`** & **`.env.example`** | File konfigurasi environment variabel seperti koneksi Postgres. |
| **`docker-compose.yml`** | Konfigurasi kontainer docker untuk database lokal dan test. |
| **`README.md`** | Dokumentasi esensial cara build, run, dan keterangan teknis proyek. |

---

## Analisis Error `ticket_units does not exist`

**Error message:**
```text
2026/08/11 17:46:33 ERROR expiry worker error err="expire held units: ERROR: relation \"ticket_units\" does not exist (SQLSTATE 42P01)"
```

**Penjelasan:**
Error di atas muncul karena program mencoba mengakses tabel `ticket_units` di database PostgreSQL melalui `CronExpiryWorker` yang diinisiasi saat server berjalan. Cron worker ini mengeksekusi query database setiap 30 detik untuk mereset tiket yang *hold*-nya expired. 

Namun, PostgreSQL melaporkan **`relation "ticket_units" does not exist`**, yang artinya **tabel tersebut belum dibuat di database**.

**Cara Memperbaikinya:**
Anda menjalankan aplikasi tanpa menjalankan skrip migrasi database terlebih dahulu. Sesuai dengan instruksi di `README.md`, Anda harus menjalankan perintah migrasi manual terlebih dahulu.

Jika server database lokal Anda berjalan via docker, jalankan perintah ini di terminal untuk membuat tabel:

```bash
psql $DATABASE_URL -f migrations/000001_create_inventory_tables.up.sql
```
*(Pastikan Anda telah menginstall `postgresql-client` atau bisa menggunakan psql di dalam docker container)*.
