package routes

import "github.com/go-chi/chi/v5"

func registerDashboard(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// ORGANIZER: lihat metrik real-time per event
		r.With(d.RequireOrganizer).Get("/api/v1/events/{eventID}/metrics", d.Dashboard.GetEventMetrics)
	})
}
