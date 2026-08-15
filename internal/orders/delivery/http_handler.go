package delivery

import (
	"crypto/hmac"
	"encoding/json"
	"errors"
	"net/http"

	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	"github.com/ebk-tech/be-booking-events/internal/orders/application"
	"github.com/ebk-tech/be-booking-events/internal/orders/domain"
	"github.com/go-chi/chi/v5"
)

type OrdersHandler struct {
	createOrderUC    *application.CreateOrderUseCase
	initiatePayUC    *application.InitiatePaymentUseCase
	confirmPayUC     *application.ConfirmPaymentUseCase
	requestRefundUC  *application.RequestRefundUseCase
	approveRefundUC  *application.ApproveRefundUseCase
	getOrderUC       *application.GetOrderUseCase
	xenditCallbackToken string
}

func NewOrdersHandler(
	createOrderUC *application.CreateOrderUseCase,
	initiatePayUC *application.InitiatePaymentUseCase,
	confirmPayUC *application.ConfirmPaymentUseCase,
	requestRefundUC *application.RequestRefundUseCase,
	approveRefundUC *application.ApproveRefundUseCase,
	getOrderUC *application.GetOrderUseCase,
	xenditCallbackToken string,
) *OrdersHandler {
	return &OrdersHandler{
		createOrderUC:       createOrderUC,
		initiatePayUC:       initiatePayUC,
		confirmPayUC:        confirmPayUC,
		requestRefundUC:     requestRefundUC,
		approveRefundUC:     approveRefundUC,
		getOrderUC:          getOrderUC,
		xenditCallbackToken: xenditCallbackToken,
	}
}

// POST /api/v1/orders
func (h *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventID string   `json:"event_id"`
		UnitIDs []string `json:"unit_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	buyerID := authdelivery.UserIDFromCtx(r.Context())
	order, err := h.createOrderUC.Execute(r.Context(), application.CreateOrderInput{
		BuyerID: buyerID,
		EventID: req.EventID,
		UnitIDs: req.UnitIDs,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNoHeldUnits) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, order)
}

// GET /api/v1/orders/{orderID}
func (h *OrdersHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	buyerID := authdelivery.UserIDFromCtx(r.Context())

	order, err := h.getOrderUC.Execute(r.Context(), orderID, buyerID)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			writeError(w, http.StatusNotFound, "order tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, order)
}

// POST /api/v1/orders/{orderID}/pay
func (h *OrdersHandler) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	buyerID := authdelivery.UserIDFromCtx(r.Context())

	out, err := h.initiatePayUC.Execute(r.Context(), orderID, buyerID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order tidak ditemukan")
		case errors.Is(err, domain.ErrOrderNotPending):
			writeError(w, http.StatusBadRequest, "order tidak bisa dibayar")
		default:
			writeError(w, http.StatusInternalServerError, "gagal membuat invoice")
		}
		return
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/v1/payments/webhook/xendit
// Tidak pakai JWT auth — diverifikasi via x-callback-token header dari Xendit.
func (h *OrdersHandler) XenditWebhook(w http.ResponseWriter, r *http.Request) {
	// Verifikasi callback token Xendit.
	// === PERUBAHAN BARU: Memperketat validasi callback token agar aman dari token kosong ===
	token := r.Header.Get("x-callback-token")
	if h.xenditCallbackToken == "" || token == "" || !hmac.Equal([]byte(token), []byte(h.xenditCallbackToken)) {
		writeError(w, http.StatusUnauthorized, "callback token tidak valid")
		return
	}
	// === AKHIR PERUBAHAN BARU ===

	var payload struct {
		ID            string `json:"id"`           // xendit invoice ID
		ExternalID    string `json:"external_id"`  // order ID kita
		Status        string `json:"status"`       // "PAID" | "EXPIRED"
		PaymentMethod string `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "payload tidak valid")
		return
	}

	err := h.confirmPayUC.Execute(r.Context(), application.ConfirmPaymentInput{
		XenditInvoiceID: payload.ID,
		ExternalID:      payload.ExternalID,
		PaymentMethod:   payload.PaymentMethod,
		Status:          payload.Status,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal konfirmasi pembayaran")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/v1/orders/{orderID}/refund — BUYER request
func (h *OrdersHandler) RequestRefund(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")
	buyerID := authdelivery.UserIDFromCtx(r.Context())

	err := h.requestRefundUC.Execute(r.Context(), orderID, buyerID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order tidak ditemukan")
		case errors.Is(err, domain.ErrOrderNotPaid):
			writeError(w, http.StatusBadRequest, "hanya order yang sudah dibayar bisa direfund")
		case errors.Is(err, domain.ErrTicketAlreadyAdmitted):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refund_requested"})
}

// POST /api/v1/orders/{orderID}/refund/approve — ORGANIZER approve
func (h *OrdersHandler) ApproveRefund(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderID")

	err := h.approveRefundUC.Execute(r.Context(), orderID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrOrderNotFound):
			writeError(w, http.StatusNotFound, "order tidak ditemukan")
		case errors.Is(err, domain.ErrRefundNotRequested):
			writeError(w, http.StatusBadRequest, "tidak ada permintaan refund untuk order ini")
		default:
			writeError(w, http.StatusInternalServerError, "gagal memproses refund")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refunded"})
}


func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
