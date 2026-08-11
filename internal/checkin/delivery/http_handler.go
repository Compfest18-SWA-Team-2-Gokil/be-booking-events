package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	"github.com/ebk-tech/be-booking-events/internal/checkin/application"
	"github.com/ebk-tech/be-booking-events/internal/checkin/domain"
)

type CheckinHandler struct {
	issueUC *application.IssueTicketUseCase
	scanUC  *application.ScanTicketUseCase
}

func NewCheckinHandler(issueUC *application.IssueTicketUseCase, scanUC *application.ScanTicketUseCase) *CheckinHandler {
	return &CheckinHandler{issueUC: issueUC, scanUC: scanUC}
}

type issueRequest struct {
	TicketUnitID string `json:"ticket_unit_id"`
}

type scanRequest struct {
	QRContent string `json:"qr_content"`
}

// POST /api/v1/checkin/issue
func (h *CheckinHandler) IssueTicket(w http.ResponseWriter, r *http.Request) {
	var req issueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TicketUnitID == "" {
		writeError(w, http.StatusBadRequest, "ticket_unit_id wajib diisi")
		return
	}

	out, err := h.issueUC.Execute(r.Context(), req.TicketUnitID)
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotConfirmed) {
			writeError(w, http.StatusNotFound, "tiket tidak ditemukan atau belum CONFIRMED")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, out)
}

// POST /api/v1/checkin/scan  (requires GATE_OPERATOR role)
// GateOperatorID diambil dari JWT context, bukan request body.
func (h *CheckinHandler) ScanTicket(w http.ResponseWriter, r *http.Request) {
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.QRContent == "" {
		writeError(w, http.StatusBadRequest, "qr_content wajib diisi")
		return
	}

	gateOperatorID := delivery.UserIDFromCtx(r.Context())
	out, err := h.scanUC.Execute(r.Context(), application.ScanTicketInput{
		QRContent:      req.QRContent,
		GateOperatorID: gateOperatorID,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidSignature), errors.Is(err, domain.ErrInvalidQRPayload):
			writeError(w, http.StatusUnprocessableEntity, "QR tidak sah")
		case errors.Is(err, domain.ErrAlreadyAdmitted):
			writeError(w, http.StatusConflict, "tiket sudah digunakan atau tidak valid")
		case errors.Is(err, domain.ErrGateOperatorNotAssigned):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
