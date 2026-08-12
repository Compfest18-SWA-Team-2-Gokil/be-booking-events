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
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

const migrationsDir = "migrations"

// ensureLogTable membuat tabel migration_log jika belum ada.
func ensureLogTable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS migration_log (
			id          SERIAL PRIMARY KEY,
			filename    TEXT NOT NULL UNIQUE,
			applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

// appliedMigrations mengembalikan set nama file yang sudah diterapkan.
func appliedMigrations(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, "SELECT filename FROM migration_log ORDER BY filename")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

// listMigrationFiles mengembalikan semua *.up.sql diurutkan ascending.
func listMigrationFiles() ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("baca direktori migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}

// listDownFiles mengembalikan semua *.down.sql diurutkan descending.
func listDownFiles() ([]string, error) {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil, fmt.Errorf("baca direktori migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".down.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files, nil
}

func runUp(ctx context.Context, conn *pgx.Conn) error {
	applied, err := appliedMigrations(ctx, conn)
	if err != nil {
		return fmt.Errorf("baca migration_log: %w", err)
	}

	files, err := listMigrationFiles()
	if err != nil {
		return err
	}

	ran := 0
	for _, f := range files {
		if applied[f] {
			slog.Info("skip (already applied)", "file", f)
			continue
		}

		sql, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("baca %s: %w", f, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("jalankan %s: %w", f, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO migration_log (filename) VALUES ($1)", f); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("catat migration_log untuk %s: %w", f, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		slog.Info("applied", "file", f)
		ran++
	}

	if ran == 0 {
		slog.Info("semua migrasi sudah up-to-date")
	} else {
		slog.Info("migrasi selesai", "applied", ran)
	}
	return nil
}

func runDown(ctx context.Context, conn *pgx.Conn) error {
	applied, err := appliedMigrations(ctx, conn)
	if err != nil {
		return fmt.Errorf("baca migration_log: %w", err)
	}

	downFiles, err := listDownFiles()
	if err != nil {
		return err
	}

	for _, df := range downFiles {
		// Cari nama .up.sql yang sesuai
		upName := strings.TrimSuffix(df, ".down.sql") + ".up.sql"
		if !applied[upName] {
			continue // belum pernah diapply, skip
		}

		sql, err := os.ReadFile(filepath.Join(migrationsDir, df))
		if err != nil {
			return fmt.Errorf("baca %s: %w", df, err)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("jalankan %s: %w", df, err)
		}

		if _, err := tx.Exec(ctx,
			"DELETE FROM migration_log WHERE filename = $1", upName); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("hapus migration_log untuk %s: %w", upName, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return err
		}

		slog.Info("rolled back", "file", df)
		return nil // hanya rollback 1 migrasi terakhir
	}

	slog.Info("tidak ada migrasi untuk di-rollback")
	return nil
}

func runStatus(ctx context.Context, conn *pgx.Conn) error {
	applied, err := appliedMigrations(ctx, conn)
	if err != nil {
		return fmt.Errorf("baca migration_log: %w", err)
	}

	files, err := listMigrationFiles()
	if err != nil {
		return err
	}

	fmt.Printf("%-55s  %s\n", "MIGRATION FILE", "STATUS")
	fmt.Println(strings.Repeat("-", 65))
	for _, f := range files {
		status := "[ ] pending"
		if applied[f] {
			status = "[✓] applied"
		}
		fmt.Printf("%-55s  %s\n", f, status)
	}
	return nil
}

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

	if err := ensureLogTable(ctx, conn); err != nil {
		slog.Error("gagal buat migration_log table", "err", err)
		os.Exit(1)
	}

	switch cmd {
	case "up":
		err = runUp(ctx, conn)
	case "down":
		err = runDown(ctx, conn)
	case "status":
		err = runStatus(ctx, conn)
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
