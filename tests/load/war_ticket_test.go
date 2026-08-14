// Package load — simulasi "war tiket": N user daftar akun lalu berebut tiket secara bersamaan.
//
// Skenario end-to-end:
//  1. Setiap goroutine register akun unik (dalam batch agar server tidak collapse)
//  2. Login → dapat JWT token
//  3. Semua menunggu di barrier (channel) — lalu serentak hit POST /tickets/hold
//  4. Laporan: siapa dapat tiket, berapa yang kehabisan, latency breakdown
//
// Cara jalankan:
//
//	TICKET_TYPE_ID=<uuid> EVENT_ID=<uuid> \
//	  go test ./tests/load/... -v -run TestWarTicket$ -timeout 180s
//
//	# 1000 user
//	RUN_1000=1 TICKET_TYPE_ID=<uuid> EVENT_ID=<uuid> \
//	  go test ./tests/load/... -v -run TestWarTicket_1000 -timeout 300s
package load

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWarTicket mensimulasikan full buyer flow: register -> login -> hold tiket.
// Semua user "dilepas" secara bersamaan setelah semua login selesai.
func TestWarTicket(t *testing.T) {
	if ticketTypeID == "" {
		t.Skip("Set TICKET_TYPE_ID env var")
	}

	users := envOrDefaultInt("CONCURRENT_USERS", 100)
	t.Logf("War Ticket: %d user simultan (register -> login -> hold)", users)

	// Phase 1: Register + Login semua user.
	t.Log("Phase 1: Register & Login...")
	tokens := make([]string, users)
	var regFail atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			email := fmt.Sprintf("waruser_%d_%d@load.test", idx, idx)
			token, err := registerAndLogin(email, "password123", fmt.Sprintf("WarUser%d", idx))
			if err != nil {
				regFail.Add(1)
				return
			}
			tokens[idx] = token
		}(i)
	}
	wg.Wait()

	if regFail.Load() > 0 {
		t.Logf("  %d user gagal register/login", regFail.Load())
	}

	validTokens := make([]string, 0, users)
	for _, tok := range tokens {
		if tok != "" {
			validTokens = append(validTokens, tok)
		}
	}
	t.Logf("Phase 1 selesai: %d/%d user siap bertempur", len(validTokens), users)

	if len(validTokens) == 0 {
		t.Fatal("tidak ada user yang berhasil login — cek server")
	}

	// Phase 2: War — semua hit hold secara bersamaan.
	t.Log("Phase 2: WAR! Semua user berebut tiket sekaligus...")

	results := make([]holdResult, len(validTokens))
	start := make(chan struct{})
	var wg2 sync.WaitGroup

	for i, tok := range validTokens {
		wg2.Add(1)
		go func(idx int, token string) {
			defer wg2.Done()
			<-start
			t0 := time.Now()
			code, err := holdTicket(ticketTypeID, token, 1)
			results[idx] = holdResult{status: code, latency: time.Since(t0), err: err}
		}(i, tok)
	}

	warStart := time.Now()
	close(start)
	wg2.Wait()
	warDuration := time.Since(warStart)

	printWarReport(t, results, len(validTokens), warDuration)
}

// TestWarTicket_1000 — 1000 user war ticket.
// Register dilakukan dalam batch 50 agar server tidak kewalahan,
// tapi hold dilakukan serentak oleh semua user.
func TestWarTicket_1000(t *testing.T) {
	if ticketTypeID == "" {
		t.Skip("Set TICKET_TYPE_ID env var")
	}
	if os.Getenv("RUN_1000") == "" {
		t.Skip("Set RUN_1000=1 untuk jalankan war ticket 1000 user (butuh stok cukup dan server siap)")
	}

	const (
		users     = 1000
		batchSize = 50
	)
	t.Logf("War Ticket 1000 user — register batch %d, hold serentak", batchSize)

	// Phase 1: Register dalam batch berurutan, tiap batch concurrent.
	tokens := make([]string, users)
	var regFail atomic.Int32

	for batchStart := 0; batchStart < users; batchStart += batchSize {
		end := batchStart + batchSize
		if end > users {
			end = users
		}
		var wg sync.WaitGroup
		for i := batchStart; i < end; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				email := fmt.Sprintf("war1k_%d@load.test", idx)
				tok, err := registerAndLogin(email, "password123", fmt.Sprintf("War%d", idx))
				if err != nil {
					regFail.Add(1)
					return
				}
				tokens[idx] = tok
			}(i)
		}
		wg.Wait()
	}

	valid := make([]string, 0, users)
	for _, tok := range tokens {
		if tok != "" {
			valid = append(valid, tok)
		}
	}
	t.Logf("Phase 1 selesai: %d/%d user siap (gagal: %d)", len(valid), users, regFail.Load())

	if len(valid) == 0 {
		t.Fatal("tidak ada user yang berhasil login")
	}

	// Phase 2: Semua hold serentak.
	t.Logf("Phase 2: WAR! %d user serentak berebut tiket...", len(valid))
	results := make([]holdResult, len(valid))
	start := make(chan struct{})
	var wg2 sync.WaitGroup

	for i, tok := range valid {
		wg2.Add(1)
		go func(idx int, token string) {
			defer wg2.Done()
			<-start
			t0 := time.Now()
			code, err := holdTicket(ticketTypeID, token, 1)
			results[idx] = holdResult{status: code, latency: time.Since(t0), err: err}
		}(i, tok)
	}

	warStart := time.Now()
	close(start)
	wg2.Wait()

	printWarReport(t, results, len(valid), time.Since(warStart))
}

// holdResult tipe untuk hasil satu goroutine.
type holdResult struct {
	status  int
	latency time.Duration
	err     error
}

func registerAndLogin(email, password, name string) (string, error) {
	regBody, _ := json.Marshal(map[string]any{
		"email": email, "name": name, "password": password,
	})
	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := sharedClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("register: %w", err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	loginBody, _ := json.Marshal(map[string]any{"email": email, "password": password})
	req2, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := sharedClient.Do(req2)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		return "", fmt.Errorf("login status %d", resp2.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&result); err != nil || result.Token == "" {
		return "", fmt.Errorf("parse token gagal")
	}
	return result.Token, nil
}

func printWarReport(t *testing.T, results []holdResult, total int, elapsed time.Duration) {
	t.Helper()
	var success, conflict, errors int32
	latencies := make([]time.Duration, 0, total)

	for _, r := range results {
		if r.err != nil {
			errors++
			continue
		}
		latencies = append(latencies, r.latency)
		switch r.status {
		case 200:
			success++
		case 409:
			conflict++
		default:
			errors++
		}
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	rps := float64(total) / elapsed.Seconds()

	t.Log("========================================")
	t.Log("  WAR TICKET REPORT")
	t.Log("----------------------------------------")
	t.Logf("  Total user      : %d", total)
	t.Logf("  Durasi war      : %s", elapsed.Round(time.Millisecond))
	t.Logf("  Throughput      : %.1f req/s", rps)
	t.Log("----------------------------------------")
	t.Logf("  Dapat tiket (200)  : %d", success)
	t.Logf("  Kehabisan (409)    : %d", conflict)
	t.Logf("  Error (timeout/5xx): %d", errors)
	if len(latencies) > 0 {
		t.Log("----------------------------------------")
		t.Logf("  Latency p50     : %s", pct(latencies, 50))
		t.Logf("  Latency p95     : %s", pct(latencies, 95))
		t.Logf("  Latency p99     : %s", pct(latencies, 99))
		t.Logf("  Latency max     : %s", latencies[len(latencies)-1])
	}
	t.Log("========================================")

	if errors > int32(total/5) {
		t.Errorf("terlalu banyak error: %d/%d — cek koneksi server atau timeout", errors, total)
	}
}
