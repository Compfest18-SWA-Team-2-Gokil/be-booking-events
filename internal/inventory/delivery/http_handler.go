package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/application"
	"github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/inventory/domain"
)

type InventoryHandler struct {
	holdUseCase *application.HoldTicketUseCase
}

func NewInventoryHandler(holdUseCase *application.HoldTicketUseCase) *InventoryHandler {
	return &InventoryHandler{holdUseCase: holdUseCase}
}

type holdItemRequest struct {
	TicketTypeID string `json:"ticket_type_id"`
	Quantity     int    `json:"quantity"`
}

type holdTicketRequest struct {
	Items []holdItemRequest `json:"items"`
}

type holdTicketResponse struct {
	HeldUntil string   `json:"held_until"`
	UnitIDs   []string `json:"unit_ids"`
}

// POST /api/v1/tickets/hold
func (h *InventoryHandler) HoldTicket(w http.ResponseWriter, r *http.Request) {
	var req holdTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	items := make([]application.HoldRequest, len(req.Items))
	for i, item := range req.Items {
		items[i] = application.HoldRequest{
			TicketTypeID: item.TicketTypeID,
			Quantity:     item.Quantity,
		}
	}

	result, err := h.holdUseCase.Execute(r.Context(), application.HoldTicketInput{Items: items})
	if err != nil {
		if errors.Is(err, domain.ErrTicketNotAvailable) {
			writeError(w, http.StatusConflict, "tiket tidak tersedia")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	unitIDs := make([]string, len(result.Units))
	for i, u := range result.Units {
		unitIDs[i] = u.ID
	}

	writeJSON(w, http.StatusOK, holdTicketResponse{
		HeldUntil: result.HeldUntil.Format("2006-01-02T15:04:05Z07:00"),
		UnitIDs:   unitIDs,
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
