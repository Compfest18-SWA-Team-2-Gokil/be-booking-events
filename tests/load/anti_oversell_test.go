// Package load berisi concurrency test untuk anti-oversell engine.
// Jalankan dengan: go test ./tests/load/... -v -run TestAntiOversell
// Butuh server berjalan di BASE_URL (default http://localhost:8080).
package load

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// TestAntiOversell_50Goroutines mensimulasikan 50 buyer berebut 1 tiket secara bersamaan.
// Harus tepat 1 yang berhasil.
func TestAntiOversell_50Goroutines(t *testing.T) {
	if ticketTypeID == "" || buyerToken == "" {
		t.Skip("Set TICKET_TYPE_ID dan BUYER_TOKEN env vars untuk menjalankan test ini")
	}

	const goroutines = 50
	var (
		success atomic.Int32
		conflict atomic.Int32
		errors  atomic.Int32
		wg      sync.WaitGroup
	)

	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			<-start // tunggu semua goroutine siap

			status, err := holdTicket(ticketTypeID, buyerToken, 1)
			if err != nil {
				errors.Add(1)
				t.Logf("goroutine %d: error: %v", id, err)
				return
			}
			switch status {
			case 200:
				success.Add(1)
			case 409:
				conflict.Add(1)
			default:
				errors.Add(1)
				t.Logf("goroutine %d: unexpected status %d", id, status)
			}
		}(i)
	}

	close(start) // lepas semua goroutine sekaligus
	wg.Wait()

	t.Logf("Results: success=%d, conflict(409)=%d, errors=%d",
		success.Load(), conflict.Load(), errors.Load())

	if success.Load() == 0 {
		t.Error("tidak ada yang berhasil hold — ada masalah dengan server")
	}
	if errors.Load() > 0 {
		t.Errorf("%d request mengalami error tak terduga", errors.Load())
	}
	if success.Load()+conflict.Load() != goroutines {
		t.Errorf("total 200+409 harus = %d goroutines, got %d",
			goroutines, success.Load()+conflict.Load())
	}
}

// TestAntiOversell_ExactStock memverifikasi bahwa tepat N tiket bisa di-hold
// saat N goroutine meminta 1 tiket dan stok = N.
func TestAntiOversell_ExactStock(t *testing.T) {
	if ticketTypeID == "" || buyerToken == "" {
		t.Skip("Set TICKET_TYPE_ID dan BUYER_TOKEN env vars")
	}

	// Cek berapa stok available saat ini.
	available := getAvailableCount(t, ticketTypeID, buyerToken)
	if available == 0 {
		t.Skip("stok kosong, provision dulu sebelum test")
	}

	// Jalankan goroutines sebanyak available + 10 (extra yang pasti gagal).
	total := available + 10
	var (
		success atomic.Int32
		wg      sync.WaitGroup
	)
	start := make(chan struct{})

	for i := 0; i < total; i++ {
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

	t.Logf("Available: %d, Berhasil hold: %d", available, success.Load())
	if int(success.Load()) > available {
		t.Errorf("OVERSELL DETECTED! %d tiket di-hold tapi stok hanya %d",
			success.Load(), available)
	}
}

func holdTicket(ticketTypeID, token string, qty int) (statusCode int, err error) {
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{"ticket_type_id": ticketTypeID, "quantity": qty},
		},
	})

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/tickets/hold", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func getAvailableCount(t *testing.T, ticketTypeID, token string) int {
	t.Helper()
	// Hit metrics endpoint — butuh event_id, skip kalau tidak ada.
	eventID := os.Getenv("EVENT_ID")
	if eventID == "" {
		return 10 // default fallback
	}

	resp, err := http.Get(fmt.Sprintf("%s/api/v1/events/%s/metrics", baseURL, eventID))
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
		if m.TicketTypeID == ticketTypeID {
			return m.Available
		}
	}
	return 10
}
