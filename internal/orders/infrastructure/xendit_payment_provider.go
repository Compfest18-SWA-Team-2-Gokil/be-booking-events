package infrastructure

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ebk-tech/be-booking-events/internal/orders/application"
)

const defaultXenditBaseURL = "https://api.xendit.co"

type XenditPaymentProvider struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

func NewXenditPaymentProvider(secretKey string) *XenditPaymentProvider {
	return &XenditPaymentProvider{
		secretKey:  secretKey,
		baseURL:    defaultXenditBaseURL,
		httpClient: &http.Client{},
	}
}

func (p *XenditPaymentProvider) getBaseURL() string {
	if p.baseURL != "" {
		return p.baseURL
	}
	return defaultXenditBaseURL
}

var _ application.PaymentProvider = (*XenditPaymentProvider)(nil)

func (p *XenditPaymentProvider) CreateInvoice(ctx context.Context, input application.CreateInvoiceInput) (*application.InvoiceResult, error) {
	body, _ := json.Marshal(map[string]any{
		"external_id":  input.ExternalID,
		"amount":       input.Amount,
		"payer_email":  input.PayerEmail,
		"description":  input.Description,
		"currency":     "IDR",
		"invoice_duration": 3600, 
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.getBaseURL()+"/v2/invoices", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.secretKey, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("xendit request gagal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var xenditErr struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&xenditErr)
		return nil, fmt.Errorf("xendit error %d: %s — %s", resp.StatusCode, xenditErr.ErrorCode, xenditErr.Message)
	}

	var result struct {
		ID         string `json:"id"`
		InvoiceURL string `json:"invoice_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode xendit response: %w", err)
	}

	return &application.InvoiceResult{
		InvoiceID:  result.ID,
		InvoiceURL: result.InvoiceURL,
	}, nil
}

func (p *XenditPaymentProvider) RefundPayment(ctx context.Context, invoiceID string, amount int64) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"invoice_id": invoiceID,
		"currency":   "IDR",
		"reason":     "CANCELLATION",
		"amount":     amount,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.getBaseURL()+"/refunds", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.secretKey, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("xendit refund request gagal: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var xenditErr struct {
			ErrorCode string `json:"error_code"`
			Message   string `json:"message"`
		}
		json.NewDecoder(resp.Body).Decode(&xenditErr)
		return "", fmt.Errorf("xendit refund error %d: %s — %s", resp.StatusCode, xenditErr.ErrorCode, xenditErr.Message)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode xendit refund response: %w", err)
	}

	return result.ID, nil
}
