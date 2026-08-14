# Dokumentasi Makefile dan Database (Migration & Seed)

Dokumen ini menjelaskan cara pengelolaan skema database (migrasi), pengisian data awal (*seeding*), serta daftar perintah pintas (*shortcuts*) yang disediakan melalui `Makefile` untuk mempermudah proses pengembangan aplikasi *Booking Events*.

---

## 1. Daftar Perintah Makefile

Aplikasi ini menggunakan `Makefile` untuk mendefinisikan perintah-perintah yang sering digunakan selama pengembangan. Berikut adalah daftar target perintah yang tersedia:

| Perintah | Command Golang | Deskripsi |
| --- | --- | --- |
| `make migrate` | `go run ./cmd/migrate up` | Menerapkan semua file migrasi database (`.up.sql`) baru yang belum dijalankan. |
| `make seed` | `go run ./cmd/seed` | Memasukkan seluruh data awal (*seeds*) untuk kebutuhan testing/development ke database. |
| `make test` | `go test ./...` | Menjalankan seluruh *unit test* dan *integration test* di dalam proyek. |
| `make run` | `go run ./cmd/server` | Menjalankan HTTP API server lokal. |
| `make setup` | `make migrate seed` | Menjalankan migrasi database diikuti dengan seeding secara berurutan. Cocok untuk inisialisasi awal. |
| `make check` | `make test` | Alias untuk menjalankan pengujian aplikasi. |

---

## 2. Migrasi Database (`cmd/migrate`)

Proses pembaruan skema database dikelola melalui program migrasi mandiri di direktori `cmd/migrate/`. Program ini membaca file SQL mentah di direktori `/migrations`.

### Struktur File Migrasi
Setiap migrasi terdiri dari sepasang file SQL dengan format penamaan `<nomor_versi>_<nama_migrasi>.<arah>.sql`:
* **`*.up.sql`**: Berisi query DDL untuk memperbarui/membuat tabel dan indeks baru (misal: `000001_create_inventory_tables.up.sql`).
* **`*.down.sql`**: Berisi query DDL untuk membatalkan perubahan yang dibuat di versi tersebut (misal: `000001_create_inventory_tables.down.sql`).

### Cara Kerja Program Migrasi
Program akan otomatis membuat tabel `migration_log` di PostgreSQL saat pertama kali dijalankan untuk mencatat riwayat migrasi yang sudah diterapkan.

Anda dapat menjalankan program migrasi dengan perintah berikut:
1. **Menerapkan Migrasi Baru (Up)**
   ```bash
   make migrate
   # atau secara manual:
   go run ./cmd/migrate up
   ```
   *Program akan membandingkan file di folder `/migrations` dengan data di tabel `migration_log`, lalu menjalankan migrasi yang belum terpasang secara berurutan (ascending).*

2. **Membatalkan Migrasi Terakhir (Down/Rollback)**
   ```bash
   go run ./cmd/migrate down
   ```
   *Program akan mengambil 1 versi migrasi terakhir yang terpasang, menjalankan file `*.down.sql` yang sesuai, lalu menghapus catatan versi tersebut dari tabel `migration_log`.*

3. **Memeriksa Status Migrasi**
   ```bash
   go run ./cmd/migrate status
   ```
   *Menampilkan daftar seluruh migrasi beserta statusnya apakah sudah terpasang (`[✓] applied`) atau masih tertunda (`[ ] pending`).*

---

## 3. Seeding Database (`cmd/seed`)

*Seeding* adalah proses pengisian database dengan data awal untuk pengembangan dan pengujian (misal: data user, event, tipe tiket). File SQL seeder berada di folder `/seeds`.

### Struktur File Seeds
File seeder berupa file `.sql` biasa yang dinamai secara berurutan:
* `seeds/01_users.sql` (menyediakan akun dengan peran `ORGANIZER`, `BUYER`, dan `GATE_OPERATOR`)
* `seeds/02_events.sql`
* `seeds/03_ticket_types.sql`
* `seeds/04_gate_operator_assignments.sql`

*Catatan: Semua user bawaan hasil seeding memiliki password default `Password123!` (tersimpan dalam bentuk Bcrypt hash).*

### Cara Kerja Program Seeding
Program seeding akan membaca file seeder, mengurutkannya secara alfabetis, lalu menjalankan setiap instruksi SQL di dalam transaksi database (`BEGIN-COMMIT`). Jika terjadi kegagalan pada salah satu file, transaksi akan dibatalkan secara keseluruhan (*rollback*).

Anda dapat mengontrol jalannya seeder dengan opsi berikut:
1. **Menjalankan Seluruh Seeder**
   ```bash
   make seed
   # atau secara manual:
   go run ./cmd/seed
   ```
2. **Menjalankan Seeder Tertentu (Filter)**
   Anda dapat memberikan filter nama file sebagai argumen program untuk menjalankan seeder secara spesifik:
   ```bash
   # Hanya menjalankan seeder user
   go run ./cmd/seed users
   
   # Hanya menjalankan seeder user dan event
   go run ./cmd/seed users events
   ```

---

## 4. Alur Kerja Awal (Workflow)

Saat memulai pengembangan dari awal atau setelah mereset database, gunakan alur berikut untuk menyiapkan lingkungan:

1. **Jalankan container database PostgreSQL & Redis (jika menggunakan docker)**
   ```bash
   docker-compose up -d
   ```
2. **Inisialisasi Skema & Data Awal**
   ```bash
   make setup
   ```
3. **Verifikasi Status Migrasi**
   ```bash
   go run ./cmd/migrate status
   ```
4. **Jalankan Automated Tests untuk Memastikan Semuanya Berfungsi**
   ```bash
   make test
   ```
5. **Nyalakan Server**
   ```bash
   make run
   ```
