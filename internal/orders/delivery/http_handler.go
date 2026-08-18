package delivery

import (
	"crypto/hmac"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	authdelivery "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/delivery"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/domain"
	"github.com/go-chi/chi/v5"
)

type OrdersHandler struct {
	createOrderUC       *application.CreateOrderUseCase
	initiatePayUC       *application.InitiatePaymentUseCase
	confirmPayUC        *application.ConfirmPaymentUseCase
	requestRefundUC     *application.RequestRefundUseCase
	approveRefundUC     *application.ApproveRefundUseCase
	getOrderUC          *application.GetOrderUseCase
	orderRepo           application.OrderRepository
	xenditCallbackToken string
}

func NewOrdersHandler(
	createOrderUC *application.CreateOrderUseCase,
	initiatePayUC *application.InitiatePaymentUseCase,
	confirmPayUC *application.ConfirmPaymentUseCase,
	requestRefundUC *application.RequestRefundUseCase,
	approveRefundUC *application.ApproveRefundUseCase,
	getOrderUC *application.GetOrderUseCase,
	orderRepo application.OrderRepository,
	xenditCallbackToken string,
) *OrdersHandler {
	return &OrdersHandler{
		createOrderUC:       createOrderUC,
		initiatePayUC:       initiatePayUC,
		confirmPayUC:        confirmPayUC,
		requestRefundUC:     requestRefundUC,
		approveRefundUC:     approveRefundUC,
		getOrderUC:          getOrderUC,
		orderRepo:           orderRepo,
		xenditCallbackToken: xenditCallbackToken,
	}
}

// POST /api/v1/orders
func (h *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EventID   string   `json:"event_id"`
		UnitIDs   []string `json:"unit_ids"`
		PromoCode string   `json:"promo_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	buyerID := authdelivery.UserIDFromCtx(r.Context())
	order, err := h.createOrderUC.Execute(r.Context(), application.CreateOrderInput{
		BuyerID:   buyerID,
		EventID:   req.EventID,
		UnitIDs:   req.UnitIDs,
		PromoCode: req.PromoCode,
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

// GET /api/v1/orders/my — semua order milik buyer yang sedang login
func (h *OrdersHandler) GetMyOrders(w http.ResponseWriter, r *http.Request) {
	buyerID := authdelivery.UserIDFromCtx(r.Context())
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	orders, total, err := h.getOrderUC.ExecuteByBuyer(r.Context(), buyerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil riwayat pesanan")
		return
	}
	if orders == nil {
		orders = []*domain.Order{}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"orders": orders,
		"pagination": map[string]any{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
	})
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
		case errors.Is(err, domain.ErrOrderCancelled):
			writeError(w, http.StatusBadRequest, "Batas waktu pembayaran telah habis. Pesanan telah dibatalkan otomatis.")
		case errors.Is(err, domain.ErrOrderNotPending):
			writeError(w, http.StatusBadRequest, "order tidak dalam status yang bisa dibayar")
		default:
			writeError(w, http.StatusInternalServerError, "gagal membuat invoice pembayaran")
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
		case errors.Is(err, domain.ErrRefundDeadlinePassed):
			writeError(w, http.StatusBadRequest, "Pengajuan refund ditolak: batas waktu pengajuan refund maksimal H-1 sebelum event dimulai.")
		case errors.Is(err, domain.ErrTicketAlreadyAdmitted):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "refund_requested"})
}

// POST /api/v1/orders/{orderID}/refund/approve — ORGANIZER approve (Tahap 1 -> diteruskan ke Admin)
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

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "organizer_approved",
		"message": "Refund telah disetujui organizer dan diteruskan ke Admin untuk final approval.",
	})
}

// GET /api/v1/orders/organizer/refunds — ORGANIZER melihat daftar pengajuan refund
func (h *OrdersHandler) ListOrganizerRefunds(w http.ResponseWriter, r *http.Request) {
	organizerID := authdelivery.UserIDFromCtx(r.Context())
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	refunds, total, err := h.orderRepo.GetRefundRequestsByOrganizer(r.Context(), organizerID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar refund")
		return
	}
	if refunds == nil {
		refunds = []*application.RefundRequestItem{}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"refunds": refunds,
		"pagination": map[string]any{
			"current_page": page,
			"per_page":     limit,
			"total_items":  total,
			"total_pages":  totalPages,
		},
	})
}


func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
