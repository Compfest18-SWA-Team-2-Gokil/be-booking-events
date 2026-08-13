// cmd/migrate/main.go
// Jalankan dengan:
//   go run ./cmd/migrate up      — apply semua migrasi yang belum diterapkan
//   go run ./cmd/migrate down    — rollback 1 migrasi terakhir
//   go run ./cmd/migrate status  — tampilkan status semua migrasi

package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ebk-tech/be-booking-events/internal/migration"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL tidak di-set")
		os.Exit(1)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		slog.Error("gagal connect ke database", "err", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	if err := migration.EnsureLogTable(ctx, conn); err != nil {
		slog.Error("gagal buat migration_log table", "err", err)
		os.Exit(1)
	}

	switch cmd {
	case "up":
		err = migration.RunUp(ctx, conn)
	case "down":
		err = migration.RunDown(ctx, conn)
	case "status":
		err = migration.RunStatus(ctx, conn)
	default:
		fmt.Printf("perintah tidak dikenal: %q\n", cmd)
		fmt.Println("gunakan: up | down | status")
		os.Exit(1)
	}

	if err != nil {
		slog.Error("migration gagal", "err", err)
		os.Exit(1)
	}
}
