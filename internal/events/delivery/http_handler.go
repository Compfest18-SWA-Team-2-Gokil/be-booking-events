package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	"github.com/ebk-tech/be-booking-events/internal/events/application"
	"github.com/ebk-tech/be-booking-events/internal/events/domain"
	"github.com/go-chi/chi/v5"
)

type EventsHandler struct {
	createEventUC      *application.CreateEventUseCase
	getEventUC         *application.GetEventUseCase
	listEventsUC       *application.ListEventsUseCase
	updateEventUC      *application.UpdateEventUseCase
	deleteEventUC      *application.DeleteEventUseCase
	createTicketTypeUC *application.CreateTicketTypeUseCase
	listTicketTypesUC  *application.ListTicketTypesUseCase
	updateTicketTypeUC *application.UpdateTicketTypeUseCase
	deleteTicketTypeUC *application.DeleteTicketTypeUseCase
	provisionUnitsUC   *application.ProvisionUnitsUseCase
	uploadImageUC      *application.UploadEventImageUseCase
}

func NewEventsHandler(
	createEventUC *application.CreateEventUseCase,
	getEventUC *application.GetEventUseCase,
	listEventsUC *application.ListEventsUseCase,
	updateEventUC *application.UpdateEventUseCase,
	deleteEventUC *application.DeleteEventUseCase,
	createTicketTypeUC *application.CreateTicketTypeUseCase,
	listTicketTypesUC *application.ListTicketTypesUseCase,
	updateTicketTypeUC *application.UpdateTicketTypeUseCase,
	deleteTicketTypeUC *application.DeleteTicketTypeUseCase,
	provisionUnitsUC *application.ProvisionUnitsUseCase,
	uploadImageUC *application.UploadEventImageUseCase,
) *EventsHandler {
	return &EventsHandler{
		createEventUC:      createEventUC,
		getEventUC:         getEventUC,
		listEventsUC:       listEventsUC,
		updateEventUC:      updateEventUC,
		deleteEventUC:      deleteEventUC,
		createTicketTypeUC: createTicketTypeUC,
		listTicketTypesUC:  listTicketTypesUC,
		updateTicketTypeUC: updateTicketTypeUC,
		deleteTicketTypeUC: deleteTicketTypeUC,
		provisionUnitsUC:   provisionUnitsUC,
		uploadImageUC:      uploadImageUC,
	}
}

// POST /api/v1/events
func (h *EventsHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Date        string `json:"date"` // RFC3339
		Location    string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	organizerID := authdelivery.UserIDFromCtx(r.Context())
	event, err := h.createEventUC.Execute(r.Context(), application.CreateEventInput{
		OrganizerID: organizerID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Date:        req.Date,
		Location:    req.Location,
	})
	if err != nil {
		writeEventValidationOrServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

// GET /api/v1/events
func (h *EventsHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	events, err := h.listEventsUC.Execute(r.Context(), application.ListEventsFilter{
		Category: q.Get("category"),
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// GET /api/v1/events/{eventID}
func (h *EventsHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	event, err := h.getEventUC.Execute(r.Context(), eventID)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, event)
}

// PUT /api/v1/events/{eventID}
func (h *EventsHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	organizerID := authdelivery.UserIDFromCtx(r.Context())

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Date        string `json:"date"` // RFC3339
		Location    string `json:"location"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	event, err := h.updateEventUC.Execute(r.Context(), application.UpdateEventInput{
		EventID:     eventID,
		OrganizerID: organizerID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Date:        req.Date,
		Location:    req.Location,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			writeEventValidationOrServerError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, event)
}

// DELETE /api/v1/events/{eventID}
func (h *EventsHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	organizerID := authdelivery.UserIDFromCtx(r.Context())

	err := h.deleteEventUC.Execute(r.Context(), eventID, organizerID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrCannotDeleteWithSoldTickets):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/events/{eventID}/ticket-types
func (h *EventsHandler) CreateTicketType(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")

	var req struct {
		Name       string `json:"name"`
		Price      int64  `json:"price"`
		Kind       string `json:"kind"`
		TotalQuota int    `json:"total_quota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	tt, err := h.createTicketTypeUC.Execute(r.Context(), application.CreateTicketTypeInput{
		EventID:    eventID,
		Name:       req.Name,
		Price:      req.Price,
		Kind:       req.Kind,
		TotalQuota: req.TotalQuota,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
			return
		}
		writeEventValidationOrServerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, tt)
}

// GET /api/v1/events/{eventID}/ticket-types
func (h *EventsHandler) ListTicketTypes(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	types, err := h.listTicketTypesUC.Execute(r.Context(), eventID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket_types": types})
}

// PUT /api/v1/events/{eventID}/ticket-types/{ticketTypeID}
func (h *EventsHandler) UpdateTicketType(w http.ResponseWriter, r *http.Request) {
	ticketTypeID := chi.URLParam(r, "ticketTypeID")
	organizerID := authdelivery.UserIDFromCtx(r.Context())

	var req struct {
		Name       string `json:"name"`
		Price      int64  `json:"price"`
		TotalQuota int    `json:"total_quota"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	tt, err := h.updateTicketTypeUC.Execute(r.Context(), application.UpdateTicketTypeInput{
		TicketTypeID: ticketTypeID,
		OrganizerID:  organizerID,
		Name:         req.Name,
		Price:        req.Price,
		TotalQuota:   req.TotalQuota,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTicketTypeNotFound):
			writeError(w, http.StatusNotFound, "ticket type tidak ditemukan")
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrPriceLocked):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, domain.ErrQuotaBelowSold):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeEventValidationOrServerError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, tt)
}

// DELETE /api/v1/events/{eventID}/ticket-types/{ticketTypeID}
func (h *EventsHandler) DeleteTicketType(w http.ResponseWriter, r *http.Request) {
	ticketTypeID := chi.URLParam(r, "ticketTypeID")
	organizerID := authdelivery.UserIDFromCtx(r.Context())

	err := h.deleteTicketTypeUC.Execute(r.Context(), ticketTypeID, organizerID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTicketTypeNotFound):
			writeError(w, http.StatusNotFound, "ticket type tidak ditemukan")
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
		case errors.Is(err, domain.ErrNotEventOrganizer):
			writeError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, domain.ErrCannotDeleteTTWithSoldTickets):
			writeError(w, http.StatusConflict, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /api/v1/events/{eventID}/ticket-types/{ticketTypeID}/provision
func (h *EventsHandler) ProvisionUnits(w http.ResponseWriter, r *http.Request) {
	ticketTypeID := chi.URLParam(r, "ticketTypeID")

	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	out, err := h.provisionUnitsUC.Execute(r.Context(), application.ProvisionUnitsInput{
		TicketTypeID: ticketTypeID,
		Quantity:     req.Quantity,
	})
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrTicketTypeNotFound):
			writeError(w, http.StatusNotFound, "ticket type tidak ditemukan")
		case errors.Is(err, domain.ErrQuotaExceeded):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeEventValidationOrServerError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// POST /api/v1/events/{eventID}/image
func (h *EventsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	organizerID := authdelivery.UserIDFromCtx(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 6*1024*1024)
	if err := r.ParseMultipartForm(5 * 1024 * 1024); err != nil {
		writeError(w, http.StatusBadRequest, "file terlalu besar atau format tidak valid")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "field 'image' wajib diisi")
		return
	}
	defer file.Close()

	buf := make([]byte, header.Size)
	if _, err := file.Read(buf); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membaca file")
		return
	}
	contentType := http.DetectContentType(buf)

	out, err := h.uploadImageUC.Execute(r.Context(), application.UploadEventImageInput{
		EventID:     eventID,
		OrganizerID: organizerID,
		Data:        buf,
		ContentType: contentType,
	})
	if err != nil {
		switch {
		case errors.Is(err, application.ErrFileTooLarge):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, application.ErrInvalidFileType):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrEventNotFound):
			writeError(w, http.StatusNotFound, "event tidak ditemukan")
		default:
			writeError(w, http.StatusInternalServerError, "gagal upload gambar")
		}
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func writeEventValidationOrServerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrLocationRequired),
		errors.Is(err, domain.ErrDateMustBeFuture),
		errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrInvalidQuota),
		errors.Is(err, domain.ErrInvalidKind),
		errors.Is(err, domain.ErrInvalidCategory):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
