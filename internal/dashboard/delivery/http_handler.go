package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/ebk-tech/be-booking-events/internal/dashboard/application"
	"github.com/go-chi/chi/v5"
)

type DashboardHandler struct {
	metricsUC *application.GetEventMetricsUseCase
}

func NewDashboardHandler(metricsUC *application.GetEventMetricsUseCase) *DashboardHandler {
	return &DashboardHandler{metricsUC: metricsUC}
}

type metricsResponse struct {
	EventID string                          `json:"event_id"`
	Metrics []application.TicketTypeMetrics `json:"metrics"`
}

// GET /api/v1/events/{eventID}/metrics
func (h *DashboardHandler) GetEventMetrics(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")

	metrics, err := h.metricsUC.Execute(r.Context(), eventID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, metricsResponse{
		EventID: eventID,
		Metrics: metrics,
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
