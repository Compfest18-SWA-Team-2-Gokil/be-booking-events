package routes

import "github.com/go-chi/chi/v5"

func registerEvents(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// ORGANIZER: manage events & ticket types
		r.With(d.RequireOrganizer).Post("/api/v1/events", d.Events.CreateEvent)
		r.With(d.RequireOrganizer).Post("/api/v1/events/{eventID}/ticket-types", d.Events.CreateTicketType)
		r.With(d.RequireOrganizer).Post("/api/v1/events/{eventID}/ticket-types/{ticketTypeID}/provision", d.Events.ProvisionUnits)

		// Public-ish: semua role authenticated bisa lihat events
		r.Get("/api/v1/events", d.Events.ListEvents)
		r.Get("/api/v1/events/{eventID}/ticket-types", d.Events.ListTicketTypes)
	})
}
