package delivery

import (
	"encoding/json"
	"errors"
	"net/http"

	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	"github.com/ebk-tech/be-booking-events/internal/events/application"
	"github.com/ebk-tech/be-booking-events/internal/events/domain"
	"github.com/go-chi/chi/v5"
)

type EventsHandler struct {
	createEventUC      *application.CreateEventUseCase
	listEventsUC       *application.ListEventsUseCase
	createTicketTypeUC *application.CreateTicketTypeUseCase
	listTicketTypesUC  *application.ListTicketTypesUseCase
	provisionUnitsUC   *application.ProvisionUnitsUseCase
	uploadImageUC      *application.UploadEventImageUseCase
}

func NewEventsHandler(
	createEventUC *application.CreateEventUseCase,
	listEventsUC *application.ListEventsUseCase,
	createTicketTypeUC *application.CreateTicketTypeUseCase,
	listTicketTypesUC *application.ListTicketTypesUseCase,
	provisionUnitsUC *application.ProvisionUnitsUseCase,
	uploadImageUC *application.UploadEventImageUseCase,
) *EventsHandler {
	return &EventsHandler{
		createEventUC:      createEventUC,
		listEventsUC:       listEventsUC,
		createTicketTypeUC: createTicketTypeUC,
		listTicketTypesUC:  listTicketTypesUC,
		provisionUnitsUC:   provisionUnitsUC,
		uploadImageUC:      uploadImageUC,
	}
}

// POST /api/v1/events
func (h *EventsHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Date     string `json:"date"` // RFC3339
		Location string `json:"location"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	organizerID := authdelivery.UserIDFromCtx(r.Context())
	event, err := h.createEventUC.Execute(r.Context(), application.CreateEventInput{
		OrganizerID: organizerID,
		Name:        req.Name,
		Date:        req.Date,
		Location:    req.Location,
	})
	if err != nil {
		writeValidationOrServerError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, event)
}

// GET /api/v1/events
func (h *EventsHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.listEventsUC.Execute(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
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
		writeValidationOrServerError(w, err)
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
		if errors.Is(err, domain.ErrTicketTypeNotFound) {
			writeError(w, http.StatusNotFound, "ticket type tidak ditemukan")
			return
		}
		if errors.Is(err, domain.ErrQuotaExceeded) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeValidationOrServerError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, out)
}

// POST /api/v1/events/{eventID}/image
func (h *EventsHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	eventID := chi.URLParam(r, "eventID")
	organizerID := authdelivery.UserIDFromCtx(r.Context())

	// Batasi ukuran request 5MB + overhead multipart.
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

	// Baca isi file dan deteksi MIME type dari 512 byte pertama.
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

func writeValidationOrServerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNameRequired),
		errors.Is(err, domain.ErrLocationRequired),
		errors.Is(err, domain.ErrDateMustBeFuture),
		errors.Is(err, domain.ErrInvalidPrice),
		errors.Is(err, domain.ErrInvalidQuota),
		errors.Is(err, domain.ErrInvalidKind):
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
