// Package load — concurrency test anti-oversell engine.
//
// Cara jalankan (butuh server running):
//
//	# 50 user (default cepat)
//	TICKET_TYPE_ID=<uuid> BUYER_TOKEN=<jwt> EVENT_ID=<uuid> \
//	  go test ./tests/load/... -v -run TestAntiOversell_Concurrent -timeout 120s
//
//	# 1000 user
//	CONCURRENT_USERS=1000 TICKET_TYPE_ID=<uuid> BUYER_TOKEN=<jwt> EVENT_ID=<uuid> \
//	  go test ./tests/load/... -v -run TestAntiOversell_Concurrent -timeout 120s
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

var (
	baseURL      = envOrDefault("BASE_URL", "http://localhost:8080")
	ticketTypeID = os.Getenv("TICKET_TYPE_ID")
	buyerToken   = os.Getenv("BUYER_TOKEN")
)

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// sharedClient pakai connection pool agar tidak buka socket baru tiap request.
var sharedClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 200,
		IdleConnTimeout:     30 * time.Second,
	},
}

// result menyimpan hasil satu goroutine.
type result struct {
	status  int
	latency time.Duration
	err     error
}

// TestAntiOversell_Concurrent adalah test utama dengan jumlah user yang bisa dikonfigurasi.
// Default 50, set CONCURRENT_USERS=1000 untuk test besar.
func TestAntiOversell_Concurrent(t *testing.T) {
	if ticketTypeID == "" || buyerToken == "" {
		t.Skip("Set TICKET_TYPE_ID dan BUYER_TOKEN env vars")
	}

	users := envOrDefaultInt("CONCURRENT_USERS", 50)
	t.Logf("▶ Menjalankan %d goroutine serentak ke POST /api/v1/tickets/hold", users)

	results := make([]result, users)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			t0 := time.Now()
			status, err := holdTicket(ticketTypeID, buyerToken, 1)
			results[idx] = result{status: status, latency: time.Since(t0), err: err}
		}(i)
	}

	close(start)
	wg.Wait()

	printReport(t, results, users)
}

// TestAntiOversell_1000Users — shortcut eksplisit untuk 1000 user.
func TestAntiOversell_1000Users(t *testing.T) {
	if ticketTypeID == "" || buyerToken == "" {
		t.Skip("Set TICKET_TYPE_ID dan BUYER_TOKEN env vars")
	}
	if os.Getenv("RUN_1000") == "" {
		t.Skip("Set RUN_1000=1 untuk jalankan test 1000 user (butuh stok cukup dan server siap)")
	}

	const users = 1000
	t.Logf("▶ Menjalankan %d goroutine serentak", users)

	results := make([]result, users)
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			t0 := time.Now()
			status, err := holdTicket(ticketTypeID, buyerToken, 1)
			results[idx] = result{status: status, latency: time.Since(t0), err: err}
		}(i)
	}

	close(start)
	wg.Wait()

	printReport(t, results, users)
}

// TestAntiOversell_NoOversell memverifikasi tidak ada oversell:
// success ≤ stok yang tersedia, tidak peduli berapa goroutine.
func TestAntiOversell_NoOversell(t *testing.T) {
	if ticketTypeID == "" || buyerToken == "" {
		t.Skip("Set TICKET_TYPE_ID dan BUYER_TOKEN env vars")
	}

	eventID := os.Getenv("EVENT_ID")
	available := getAvailableCount(t, eventID, ticketTypeID)
	if available == 0 {
		t.Skip("stok 0 — provision dulu sebelum test")
	}

	// Spawn available+50 goroutine supaya ada yang pasti gagal.
	users := available + 50
	if users > 1000 {
		users = 1000
	}
	t.Logf("▶ Stok=%d, goroutine=%d — memverifikasi tidak ada oversell", available, users)

	var success atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, _ := holdTicket(ticketTypeID, buyerToken, 1)
			if status == 200 {
				success.Add(1)
			}
		}()
	}

	close(start)
	wg.Wait()

	got := int(success.Load())
	t.Logf("Hasil: success=%d / stok=%d", got, available)

	if got > available {
		t.Errorf("🚨 OVERSELL DETECTED! hold berhasil %d tapi stok hanya %d", got, available)
	} else {
		t.Logf("✅ Anti-oversell OK: %d hold berhasil dari stok %d", got, available)
	}
}

// TestConcurrent_HealthCheck — sanity check: server up dan bisa terima 1000 req bersamaan.
func TestConcurrent_HealthCheck(t *testing.T) {
	users := envOrDefaultInt("CONCURRENT_USERS", 1000)
	t.Logf("▶ Health check %d concurrent GET /docs", users)

	var ok, fail atomic.Int32
	var wg sync.WaitGroup
	latencies := make([]time.Duration, users)
	start := make(chan struct{})

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			t0 := time.Now()
			resp, err := sharedClient.Get(baseURL + "/docs/openapi.yaml")
			latencies[idx] = time.Since(t0)
			if err != nil || resp.StatusCode != 200 {
				fail.Add(1)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			ok.Add(1)
		}(i)
	}

	close(start)
	wg.Wait()

	lats := latencies
	sort.Slice(lats, func(i, j int) bool { return lats[i] < lats[j] })

	t.Logf("Users: %d | OK: %d | Fail: %d", users, ok.Load(), fail.Load())
	t.Logf("Latency p50=%-8s p95=%-8s p99=%-8s max=%s",
		pct(lats, 50), pct(lats, 95), pct(lats, 99), lats[len(lats)-1])

	if fail.Load() > int32(users/10) {
		t.Errorf("terlalu banyak failure: %d/%d", fail.Load(), users)
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

func printReport(t *testing.T, results []result, total int) {
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
			t.Logf("  unexpected HTTP %d", r.status)
		}
	}

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	t.Logf("─────────────────────────────────────")
	t.Logf("Total goroutine : %d", total)
	t.Logf("Hold berhasil   : %d (HTTP 200)", success)
	t.Logf("Stok habis      : %d (HTTP 409 — expected)", conflict)
	t.Logf("Error           : %d", errors)
	if len(latencies) > 0 {
		t.Logf("Latency p50     : %s", pct(latencies, 50))
		t.Logf("Latency p95     : %s", pct(latencies, 95))
		t.Logf("Latency p99     : %s", pct(latencies, 99))
		t.Logf("Latency max     : %s", latencies[len(latencies)-1])
	}
	t.Logf("─────────────────────────────────────")

	if success == 0 {
		t.Error("tidak ada yang berhasil hold — cek server dan stok")
	}
	if errors > 0 {
		t.Errorf("%d request error tak terduga (timeout/5xx)", errors)
	}
	if int(success+conflict) != total-int(errors) {
		t.Errorf("total 200+409 (%d) + errors (%d) harus = %d goroutine",
			success+conflict, errors, total)
	}
}

func pct(sorted []time.Duration, p int) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * float64(p) / 100.0)
	return sorted[idx].Round(time.Millisecond)
}

func envOrDefaultInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n <= 0 {
		return def
	}
	return n
}

func holdTicket(ticketTypeID, token string, qty int) (statusCode int, err error) {
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{"ticket_type_id": ticketTypeID, "quantity": qty},
		},
	})

	req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/tickets/hold", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := sharedClient.Do(req)
	if err != nil {
		return 0, err
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode, nil
}

func getAvailableCount(t *testing.T, eventID, ttID string) int {
	t.Helper()
	if eventID == "" {
		return 10
	}
	resp, err := sharedClient.Get(fmt.Sprintf("%s/api/v1/events/%s/metrics", baseURL, eventID))
	if err != nil || resp.StatusCode != 200 {
		return 10
	}
	defer resp.Body.Close()

	var result struct {
		Metrics []struct {
			TicketTypeID string `json:"ticket_type_id"`
			Available    int    `json:"available"`
		} `json:"metrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 10
	}
	for _, m := range result.Metrics {
		if m.TicketTypeID == ttID {
			return m.Available
		}
	}
	return 10
}
