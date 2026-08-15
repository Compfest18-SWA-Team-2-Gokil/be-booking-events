package infrastructure

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebk-tech/be-booking-events/internal/orders/application"
)

func TestXenditPaymentProvider_CreateInvoice_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v2/invoices" {
			t.Errorf("path = %s, want /v2/invoices", r.URL.Path)
		}

		username, _, ok := r.BasicAuth()
		if !ok {
			t.Fatal("basic auth tidak ditemukan")
		}
		if username != "test-secret-key" {
			t.Errorf("basic auth username = %s, want test-secret-key", username)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["external_id"] != "order-123" {
			t.Errorf("external_id = %v, want order-123", body["external_id"])
		}
		if body["amount"] != float64(150000) {
			t.Errorf("amount = %v, want 150000", body["amount"])
		}
		if body["payer_email"] != "buyer@test.com" {
			t.Errorf("payer_email = %v, want buyer@test.com", body["payer_email"])
		}
		if body["currency"] != "IDR" {
			t.Errorf("currency = %v, want IDR", body["currency"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":          "xendit-inv-abc",
			"invoice_url": "https://checkout.xendit.co/pay/xendit-inv-abc",
		})
	}))
	defer srv.Close()

	provider := &XenditPaymentProvider{
		secretKey:  "test-secret-key",
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	result, err := provider.CreateInvoice(context.Background(), application.CreateInvoiceInput{
		ExternalID:  "order-123",
		Amount:      150000,
		PayerEmail:  "buyer@test.com",
		Description: "Pembelian tiket event event-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.InvoiceID != "xendit-inv-abc" {
		t.Errorf("invoice_id = %s, want xendit-inv-abc", result.InvoiceID)
	}
	if result.InvoiceURL != "https://checkout.xendit.co/pay/xendit-inv-abc" {
		t.Errorf("invoice_url = %s, want https://checkout.xendit.co/pay/xendit-inv-abc", result.InvoiceURL)
	}
}

func TestXenditPaymentProvider_CreateInvoice_XenditError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error_code": "API_VALIDATION_ERROR",
			"message":    "amount is required",
		})
	}))
	defer srv.Close()

	provider := &XenditPaymentProvider{
		secretKey:  "test-key",
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	_, err := provider.CreateInvoice(context.Background(), application.CreateInvoiceInput{
		ExternalID: "order-1",
		Amount:     0,
	})
	if err == nil {
		t.Fatal("expected error dari Xendit 400, got nil")
	}
}

func TestXenditPaymentProvider_RefundPayment_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/refunds" {
			t.Errorf("path = %s, want /refunds", r.URL.Path)
		}

		username, _, ok := r.BasicAuth()
		if !ok {
			t.Fatal("basic auth tidak ditemukan")
		}
		if username != "test-secret-key" {
			t.Errorf("basic auth username = %s, want test-secret-key", username)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["invoice_id"] != "xendit-inv-abc" {
			t.Errorf("invoice_id = %v, want xendit-inv-abc", body["invoice_id"])
		}
		if body["amount"] != float64(150000) {
			t.Errorf("amount = %v, want 150000", body["amount"])
		}
		if body["currency"] != "IDR" {
			t.Errorf("currency = %v, want IDR", body["currency"])
		}
		if body["reason"] != "CANCELLATION" {
			t.Errorf("reason = %v, want CANCELLATION", body["reason"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "xendit-refund-xyz",
		})
	}))
	defer srv.Close()

	provider := &XenditPaymentProvider{
		secretKey:  "test-secret-key",
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	refundID, err := provider.RefundPayment(context.Background(), "xendit-inv-abc", 150000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refundID != "xendit-refund-xyz" {
		t.Errorf("refund_id = %s, want xendit-refund-xyz", refundID)
	}
}

func TestXenditPaymentProvider_RefundPayment_XenditError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]any{
			"error_code": "REFUND_NOT_ALLOWED",
			"message":    "invoice cannot be refunded",
		})
	}))
	defer srv.Close()

	provider := &XenditPaymentProvider{
		secretKey:  "test-key",
		baseURL:    srv.URL,
		httpClient: srv.Client(),
	}

	_, err := provider.RefundPayment(context.Background(), "xendit-inv-abc", 150000)
	if err == nil {
		t.Fatal("expected error dari Xendit 422, got nil")
	}
}
