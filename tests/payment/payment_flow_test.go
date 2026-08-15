// Package payment — end-to-end payment flow simulation tests.
//
// Simulasi full lifecycle: register → login → hold → create order → initiate payment
// → xendit webhook (PAID) → request refund → approve refund.
//
// Butuh server running + Xendit sandbox credentials.
//
// Prasyarat:
//   - docker-compose up -d (postgres, redis, minio)
//   - Server running (go run ./cmd/server)
//   - Event + ticket type + stok sudah di-provision
//
// Cara jalankan:
//
//	EVENT_ID=<uuid> TICKET_TYPE_ID=<uuid> \
//	  go test ./tests/payment/... -v -run TestPaymentFlow_Full -timeout 120s
//
//	Custom server URL:
//	BASE_URL=http://localhost:8080 EVENT_ID=<uuid> TICKET_TYPE_ID=<uuid> \
//	  go test ./tests/payment/... -v -run TestPaymentFlow_Full -timeout 120s
package payment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

var (
	paymentBaseURL = envOrDefault("BASE_URL", "http://localhost:8080")
	eventID        = os.Getenv("EVENT_ID")
	ticketTypeID   = os.Getenv("TICKET_TYPE_ID")
	callbackToken  = os.Getenv("XENDIT_CALLBACK_TOKEN")
	organizerToken = os.Getenv("ORGANIZER_TOKEN")
)

var paymentClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     30 * time.Second,
	},
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b)
}

// TestPaymentFlow_Full mensimulasikan siklus hidup pembayaran lengkap:
//  1. Register buyer
//  2. Login → JWT (buyer)
//  3. Hold tiket
//  4. Create order
//  5. Initiate payment (dapat invoice URL + xendit invoice ID)
//  6. Simulate Xendit webhook PAID
//  7. Verify order status = PAID
//  8. Request refund (buyer)
//  9. Approve refund (organizer)
//  10. Verify order status = REFUNDED
func TestPaymentFlow_Full(t *testing.T) {
	if eventID == "" || ticketTypeID == "" {
		t.Skip("Set EVENT_ID dan TICKET_TYPE_ID env vars")
	}
	if callbackToken == "" {
		t.Skip("Set XENDIT_CALLBACK_TOKEN env var (cek .env)")
	}
	if organizerToken == "" {
		t.Skip("Set ORGANIZER_TOKEN env var (login organizer@compfest.id untuk dapat JWT)")
	}

	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("payflow_%d@load.test", suffix)
	name := fmt.Sprintf("PayFlow%d", suffix)
	password := "password123"

	t.Log("Step 1: Register buyer...")
	if err := registerUser(email, name, password, "BUYER"); err != nil {
		t.Fatalf("register gagal: %v", err)
	}

	t.Log("Step 2: Login buyer...")
	token, err := loginUser(email, password)
	if err != nil {
		t.Fatalf("login gagal: %v", err)
	}

	t.Log("Step 3: Hold tiket...")
	unitIDs, err := holdTickets(ticketTypeID, token, 1)
	if err != nil {
		t.Fatalf("hold gagal: %v", err)
	}
	t.Logf("  held units: %v", unitIDs)

	t.Log("Step 4: Create order...")
	orderID, err := createOrder(eventID, unitIDs, token)
	if err != nil {
		t.Fatalf("create order gagal: %v", err)
	}
	t.Logf("  order_id: %s", orderID)

	t.Log("Step 5: Initiate payment...")
	xenditInvoiceID, invoiceURL, err := initiatePayment(orderID, token)
	if err != nil {
		t.Fatalf("initiate payment gagal: %v", err)
	}
	t.Logf("  xendit_invoice_id: %s", xenditInvoiceID)
	t.Logf("  invoice_url: %s", invoiceURL)

	t.Log("Step 6: Simulate Xendit webhook PAID...")
	if err := simulateWebhookPaid(xenditInvoiceID, orderID, "CREDIT_CARD"); err != nil {
		t.Fatalf("webhook PAID gagal: %v", err)
	}

	t.Log("Step 7: Verify order status = PAID...")
	status, err := getOrderStatus(orderID, token)
	if err != nil {
		t.Fatalf("get order gagal: %v", err)
	}
	if status != "PAID" {
		t.Errorf("order status = %s, want PAID", status)
	}

	t.Log("Step 8: Request refund (buyer)...")
	if err := requestRefund(orderID, token); err != nil {
		t.Fatalf("request refund gagal: %v", err)
	}

	t.Log("Step 9: Approve refund (organizer)...")
	if err := approveRefund(orderID, organizerToken); err != nil {
		t.Fatalf("approve refund gagal: %v", err)
	}

	t.Log("Step 10: Verify order status = REFUNDED...")
	status, err = getOrderStatus(orderID, token)
	if err != nil {
		t.Fatalf("get order gagal: %v", err)
	}
	if status != "REFUNDED" {
		t.Errorf("order status = %s, want REFUNDED", status)
	}

	t.Log("Full payment flow selesai!")
}

// TestPaymentFlow_ManualConfirm sama seperti TestPaymentFlow_Full
// tapi hanya sampai webhook PAID — cocok untuk verifikasi cepat tanpa refund.
func TestPaymentFlow_ManualConfirm(t *testing.T) {
	if eventID == "" || ticketTypeID == "" {
		t.Skip("Set EVENT_ID dan TICKET_TYPE_ID env vars")
	}
	if callbackToken == "" {
		t.Skip("Set XENDIT_CALLBACK_TOKEN env var (cek .env)")
	}

	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("paymanual_%d@load.test", suffix)
	name := fmt.Sprintf("PayManual%d", suffix)
	password := "password123"

	t.Log("Register + Login...")
	if err := registerUser(email, name, password, "BUYER"); err != nil {
		t.Fatalf("register: %v", err)
	}
	token, err := loginUser(email, password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	t.Log("Hold → Create Order → Initiate Payment...")
	unitIDs, err := holdTickets(ticketTypeID, token, 1)
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	orderID, err := createOrder(eventID, unitIDs, token)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	xenditInvoiceID, _, err := initiatePayment(orderID, token)
	if err != nil {
		t.Fatalf("initiate payment: %v", err)
	}

	t.Log("Simulate webhook PAID...")
	if err := simulateWebhookPaid(xenditInvoiceID, orderID, "BANK_TRANSFER"); err != nil {
		t.Fatalf("webhook: %v", err)
	}

	status, err := getOrderStatus(orderID, token)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if status != "PAID" {
		t.Errorf("status = %s, want PAID", status)
	}
	t.Log("Manual confirm flow OK!")
}

// TestPaymentFlow_ExpiredPayment mensimulasikan webhook EXPIRED → order CANCELLED.
func TestPaymentFlow_ExpiredPayment(t *testing.T) {
	if eventID == "" || ticketTypeID == "" {
		t.Skip("Set EVENT_ID dan TICKET_TYPE_ID env vars")
	}
	if callbackToken == "" {
		t.Skip("Set XENDIT_CALLBACK_TOKEN env var (cek .env)")
	}

	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("payexpired_%d@load.test", suffix)
	name := fmt.Sprintf("PayExpired%d", suffix)
	password := "password123"

	t.Log("Register + Login...")
	if err := registerUser(email, name, password, "BUYER"); err != nil {
		t.Fatalf("register: %v", err)
	}
	token, err := loginUser(email, password)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	t.Log("Hold → Create Order → Initiate Payment...")
	unitIDs, err := holdTickets(ticketTypeID, token, 1)
	if err != nil {
		t.Fatalf("hold: %v", err)
	}
	orderID, err := createOrder(eventID, unitIDs, token)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	xenditInvoiceID, _, err := initiatePayment(orderID, token)
	if err != nil {
		t.Fatalf("initiate payment: %v", err)
	}

	t.Log("Simulate webhook EXPIRED...")
	if err := simulateWebhookExpired(xenditInvoiceID, orderID); err != nil {
		t.Fatalf("webhook EXPIRED: %v", err)
	}

	status, err := getOrderStatus(orderID, token)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if status != "CANCELLED" {
		t.Errorf("status = %s, want CANCELLED", status)
	}
	t.Log("Expired payment flow OK!")
}

// TestPaymentFlow_ConcurrentOrders mensimulasikan beberapa buyer
// yang melakukan payment flow secara bersamaan.
func TestPaymentFlow_ConcurrentOrders(t *testing.T) {
	if eventID == "" || ticketTypeID == "" {
		t.Skip("Set EVENT_ID dan TICKET_TYPE_ID env vars")
	}
	if callbackToken == "" {
		t.Skip("Set XENDIT_CALLBACK_TOKEN env var (cek .env)")
	}

	users := 5
	t.Logf("Concurrent payment flow: %d user", users)

	type flowResult struct {
		email   string
		orderID string
		status  string
		err     error
	}

	results := make([]flowResult, users)
	var wg sync.WaitGroup

	for i := 0; i < users; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			suffix := time.Now().UnixNano()
			email := fmt.Sprintf("payconc%d_%d@load.test", idx, suffix)
			name := fmt.Sprintf("PayConc%d_%d", idx, suffix)
			password := "password123"

			if err := registerUser(email, name, password, "BUYER"); err != nil {
				results[idx] = flowResult{email: email, err: fmt.Errorf("register: %w", err)}
				return
			}
			token, err := loginUser(email, password)
			if err != nil {
				results[idx] = flowResult{email: email, err: fmt.Errorf("login: %w", err)}
				return
			}

			unitIDs, err := holdTickets(ticketTypeID, token, 1)
			if err != nil {
				results[idx] = flowResult{email: email, err: fmt.Errorf("hold: %w", err)}
				return
			}

			orderID, err := createOrder(eventID, unitIDs, token)
			if err != nil {
				results[idx] = flowResult{email: email, err: fmt.Errorf("create order: %w", err)}
				return
			}

			xenditInvoiceID, _, err := initiatePayment(orderID, token)
			if err != nil {
				results[idx] = flowResult{email: email, orderID: orderID, err: fmt.Errorf("initiate payment: %w", err)}
				return
			}

			if err := simulateWebhookPaid(xenditInvoiceID, orderID, "CREDIT_CARD"); err != nil {
				results[idx] = flowResult{email: email, orderID: orderID, err: fmt.Errorf("webhook: %w", err)}
				return
			}

			status, err := getOrderStatus(orderID, token)
			if err != nil {
				results[idx] = flowResult{email: email, orderID: orderID, err: fmt.Errorf("get order: %w", err)}
				return
			}

			results[idx] = flowResult{email: email, orderID: orderID, status: status}
		}(i)
	}

	wg.Wait()

	var ok, fail int
	for i, r := range results {
		if r.err != nil {
			fail++
			t.Logf("  user %d (%s): ERROR — %v", i, r.email, r.err)
		} else if r.status == "PAID" {
			ok++
			t.Logf("  user %d (%s): order %s → PAID", i, r.email, r.orderID)
		} else {
			fail++
			t.Logf("  user %d (%s): order %s → unexpected status %s", i, r.email, r.orderID, r.status)
		}
	}

	t.Logf("Concurrent payment: %d OK, %d fail dari %d user", ok, fail, users)
	if ok == 0 {
		t.Error("tidak ada yang berhasil — cek server dan stok")
	}
}

// ── HTTP helpers ────────────────────────────────────────────────────────────

func registerUser(email, name, password, role string) error {
	body, _ := json.Marshal(map[string]any{
		"email": email, "name": name, "password": password, "role": role,
	})
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := paymentClient.Do(req)
	if err != nil {
		return err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 201 && resp.StatusCode != 200 {
		return fmt.Errorf("register status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func loginUser(email, password string) (string, error) {
	body, _ := json.Marshal(map[string]any{"email": email, "password": password})
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := paymentClient.Do(req)
	if err != nil {
		return "", err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("login status %d: %s", resp.StatusCode, respBody)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err != nil || result.Token == "" {
		return "", fmt.Errorf("parse token gagal: body=%s", respBody)
	}
	return result.Token, nil
}

func holdTickets(ttID, token string, qty int) ([]string, error) {
	body, _ := json.Marshal(map[string]any{
		"items": []map[string]any{
			{"ticket_type_id": ttID, "quantity": qty},
		},
	})
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/tickets/hold", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return nil, err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("hold status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		UnitIDs []string `json:"unit_ids"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err != nil {
		return nil, fmt.Errorf("decode hold response: %w (body=%s)", err, respBody)
	}
	return result.UnitIDs, nil
}

func createOrder(evID string, unitIDs []string, token string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"event_id": evID, "unit_ids": unitIDs,
	})
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return "", err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 201 {
		return "", fmt.Errorf("create order status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err != nil {
		return "", fmt.Errorf("decode order response: %w (body=%s)", err, respBody)
	}
	return result.ID, nil
}

func initiatePayment(orderID, token string) (xenditInvoiceID, invoiceURL string, err error) {
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/orders/"+orderID+"/pay", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return "", "", err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("initiate payment status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		PaymentID       string `json:"payment_id"`
		XenditInvoiceID string `json:"xendit_invoice_id"`
		InvoiceURL      string `json:"invoice_url"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err != nil {
		return "", "", fmt.Errorf("decode payment response: %w (body=%s)", err, respBody)
	}

	return result.XenditInvoiceID, result.InvoiceURL, nil
}

func simulateWebhookPaid(xenditInvoiceID, orderID, paymentMethod string) error {
	payload, _ := json.Marshal(map[string]any{
		"id":             xenditInvoiceID,
		"external_id":    orderID,
		"status":         "PAID",
		"payment_method": paymentMethod,
	})
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/payments/webhook/xendit", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-callback-token", callbackToken)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("webhook status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func simulateWebhookExpired(xenditInvoiceID, orderID string) error {
	payload, _ := json.Marshal(map[string]any{
		"id":          xenditInvoiceID,
		"external_id": orderID,
		"status":      "EXPIRED",
	})
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/payments/webhook/xendit", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-callback-token", callbackToken)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("webhook status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func getOrderStatus(orderID, token string) (string, error) {
	req, _ := http.NewRequest(http.MethodGet, paymentBaseURL+"/api/v1/orders/"+orderID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return "", err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("get order status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(respBody), &result); err != nil {
		return "", fmt.Errorf("decode order: %w (body=%s)", err, respBody)
	}
	return result.Status, nil
}

func requestRefund(orderID, token string) error {
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/orders/"+orderID+"/refund", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("request refund status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}

func approveRefund(orderID, organizerTok string) error {
	req, _ := http.NewRequest(http.MethodPost, paymentBaseURL+"/api/v1/orders/"+orderID+"/refund/approve", nil)
	req.Header.Set("Authorization", "Bearer "+organizerTok)

	resp, err := paymentClient.Do(req)
	if err != nil {
		return err
	}
	respBody := readBody(resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("approve refund status %d: %s", resp.StatusCode, respBody)
	}
	return nil
}
