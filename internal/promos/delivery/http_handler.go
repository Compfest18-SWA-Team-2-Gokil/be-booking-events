package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/promos/domain"
	"github.com/go-chi/chi/v5"
)

type PromoHandler struct {
	adminUC    *application.AdminPromosUseCase
	validateUC *application.ValidatePromoUseCase
}

func NewPromoHandler(adminUC *application.AdminPromosUseCase, validateUC *application.ValidatePromoUseCase) *PromoHandler {
	return &PromoHandler{
		adminUC:    adminUC,
		validateUC: validateUC,
	}
}

// GET /api/v1/admin/promos
func (h *PromoHandler) AdminListPromos(w http.ResponseWriter, r *http.Request) {
	promos, err := h.adminUC.ListAllPromos(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar promo")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promos": promos})
}

// POST /api/v1/admin/promos
func (h *PromoHandler) AdminCreatePromo(w http.ResponseWriter, r *http.Request) {
	var input application.CreatePromoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	promo, err := h.adminUC.CreatePromo(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidPromoCode) || errors.Is(err, domain.ErrInvalidDiscountValue) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal membuat promo: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, promo)
}

// PUT /api/v1/admin/promos/{id}
func (h *PromoHandler) AdminUpdatePromo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "promo id wajib diisi")
		return
	}

	var input application.UpdatePromoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	promo, err := h.adminUC.UpdatePromo(r.Context(), id, input)
	if err != nil {
		if errors.Is(err, domain.ErrPromoNotFound) {
			writeError(w, http.StatusNotFound, "promo tidak ditemukan")
			return
		}
		if errors.Is(err, domain.ErrInvalidPromoCode) || errors.Is(err, domain.ErrInvalidDiscountValue) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal memperbarui promo: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, promo)
}

// DELETE /api/v1/admin/promos/{id}
func (h *PromoHandler) AdminDeletePromo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "promo id wajib diisi")
		return
	}

	if err := h.adminUC.DeletePromo(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menghapus promo")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "promo berhasil dihapus"})
}

// POST /api/v1/promos/validate
func (h *PromoHandler) ValidatePromo(w http.ResponseWriter, r *http.Request) {
	var input application.ValidatePromoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	out, err := h.validateUC.Execute(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPromoNotFound):
			writeError(w, http.StatusNotFound, "Kode promo tidak ditemukan")
		case errors.Is(err, domain.ErrPromoInactive):
			writeError(w, http.StatusBadRequest, "Kode promo sudah tidak aktif")
		case errors.Is(err, domain.ErrPromoUsageLimitExceed):
			writeError(w, http.StatusBadRequest, "Kuota penggunaan promo telah habis")
		case errors.Is(err, domain.ErrPromoMinOrderNotMet):
			writeError(w, http.StatusBadRequest, "Nominal pesanan belum memenuhi syarat minimal promo")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, out)
}

// GET /api/v1/promos/active
func (h *PromoHandler) ListActivePromos(w http.ResponseWriter, r *http.Request) {
	promos, err := h.adminUC.ListActivePromos(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil promo aktif")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"promos": promos})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
