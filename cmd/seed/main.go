// cmd/seed/main.go
// Jalankan dengan:
//   go run ./cmd/seed              — jalankan semua seeder
//   go run ./cmd/seed users        — jalankan seeder tertentu saja
//   go run ./cmd/seed users events — jalankan beberapa seeder

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

const seedsDir = "seeds"

func listSeedFiles(filter []string) ([]string, error) {
	entries, err := os.ReadDir(seedsDir)
	if err != nil {
		return nil, fmt.Errorf("baca direktori seeds: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		// Jika ada filter, hanya include yang namanya match
		if len(filter) > 0 {
			matched := false
			for _, f := range filter {
				// Match by prefix atau nama file (tanpa .sql)
				base := strings.TrimSuffix(e.Name(), ".sql")
				if strings.Contains(base, f) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)
	return files, nil
}

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL tidak di-set")
		os.Exit(1)
	}

	// Argumen setelah nama program dianggap sebagai filter nama seeder
	filter := os.Args[1:]

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		slog.Error("gagal connect ke database", "err", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	files, err := listSeedFiles(filter)
	if err != nil {
		slog.Error("gagal baca seeds", "err", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		if len(filter) > 0 {
			slog.Warn("tidak ada seed file yang cocok dengan filter", "filter", filter)
		} else {
			slog.Warn("folder seeds/ kosong")
		}
		return
	}

	ran := 0
	for _, f := range files {
		sql, err := os.ReadFile(filepath.Join(seedsDir, f))
		if err != nil {
			slog.Error("gagal baca seed file", "file", f, "err", err)
			os.Exit(1)
		}

		tx, err := conn.Begin(ctx)
		if err != nil {
			slog.Error("gagal mulai transaksi", "err", err)
			os.Exit(1)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			slog.Error("seed gagal", "file", f, "err", err)
			os.Exit(1)
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("commit gagal", "file", f, "err", err)
			os.Exit(1)
		}

		slog.Info("seeded", "file", f)
		ran++
	}

	slog.Info("seeding selesai", "total", ran)
}
