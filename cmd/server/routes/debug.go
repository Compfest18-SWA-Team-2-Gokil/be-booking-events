package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

var paymentClient = &http.Client{
	Timeout: 15 * time.Second,
}

func readBody(resp *http.Response) string {
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return string(b)
}

func registerDebug(r chi.Router, d Deps) {
	r.Get("/api/v1/payment-detail-generate", func(w http.ResponseWriter, req *http.Request) {
		eventID := os.Getenv("EVENT_ID")
		ticketTypeID := os.Getenv("TICKET_TYPE_ID")

		if eventID == "" || ticketTypeID == "" {
			http.Error(w, "EVENT_ID and TICKET_TYPE_ID environment variables must be set", http.StatusInternalServerError)
			return
		}

		baseURL := "http://localhost:8080"
		if envHost := os.Getenv("BASE_URL"); envHost != "" {
			baseURL = envHost
		}

		suffix := time.Now().UnixNano()
		email := fmt.Sprintf("debug_pay_%d@load.test", suffix)
		name := fmt.Sprintf("DebugPay%d", suffix)
		password := "password123"

		// 1. Register
		regBody, _ := json.Marshal(map[string]any{"email": email, "name": name, "password": password, "role": "BUYER"})
		regReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/register", bytes.NewReader(regBody))
		regReq.Header.Set("Content-Type", "application/json")
		regResp, err := paymentClient.Do(regReq)
		if err != nil || (regResp.StatusCode != 201 && regResp.StatusCode != 200) {
			http.Error(w, fmt.Sprintf("Register failed: %v", err), http.StatusInternalServerError)
			return
		}

		// 2. Login
		logBody, _ := json.Marshal(map[string]any{"email": email, "password": password})
		logReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(logBody))
		logReq.Header.Set("Content-Type", "application/json")
		logResp, err := paymentClient.Do(logReq)
		if err != nil || logResp.StatusCode != 200 {
			http.Error(w, fmt.Sprintf("Login failed: %v", err), http.StatusInternalServerError)
			return
		}
		var loginRes struct{ Token string `json:"token"` }
		json.Unmarshal([]byte(readBody(logResp)), &loginRes)

		// 3. Hold Ticket
		holdBody, _ := json.Marshal(map[string]any{"items": []map[string]any{{"ticket_type_id": ticketTypeID, "quantity": 1}}})
		holdReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/tickets/hold", bytes.NewReader(holdBody))
		holdReq.Header.Set("Content-Type", "application/json")
		holdReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
		holdResp, err := paymentClient.Do(holdReq)
		if err != nil || holdResp.StatusCode != 200 {
			http.Error(w, fmt.Sprintf("Hold ticket failed: %v, body: %s", err, readBody(holdResp)), http.StatusInternalServerError)
			return
		}
		var holdRes struct{ UnitIDs []string `json:"unit_ids"` }
		json.Unmarshal([]byte(readBody(holdResp)), &holdRes)

		// 4. Create Order
		orderBody, _ := json.Marshal(map[string]any{"event_id": eventID, "unit_ids": holdRes.UnitIDs})
		orderReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/orders", bytes.NewReader(orderBody))
		orderReq.Header.Set("Content-Type", "application/json")
		orderReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
		orderResp, err := paymentClient.Do(orderReq)
		if err != nil || orderResp.StatusCode != 201 {
			http.Error(w, fmt.Sprintf("Create order failed: %v, body: %s", err, readBody(orderResp)), http.StatusInternalServerError)
			return
		}
		var orderRes struct{ ID string `json:"id"` }
		json.Unmarshal([]byte(readBody(orderResp)), &orderRes)

		// 5. Initiate Payment
		payReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/v1/orders/"+orderRes.ID+"/pay", nil)
		payReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
		payResp, err := paymentClient.Do(payReq)
		if err != nil || payResp.StatusCode != 200 {
			http.Error(w, fmt.Sprintf("Initiate payment failed: %v, body: %s", err, readBody(payResp)), http.StatusInternalServerError)
			return
		}
		var payRes struct {
			PaymentID       string `json:"payment_id"`
			XenditInvoiceID string `json:"xendit_invoice_id"`
			InvoiceURL      string `json:"invoice_url"`
		}
		json.Unmarshal([]byte(readBody(payResp)), &payRes)

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"message": "Payment simulation data generated successfully",
			"data": map[string]any{
				"email":             email,
				"password":          password,
				"order_id":          orderRes.ID,
				"payment_id":        payRes.PaymentID,
				"xendit_invoice_id": payRes.XenditInvoiceID,
				"invoice_url":       payRes.InvoiceURL,
			},
		})
	})
}
