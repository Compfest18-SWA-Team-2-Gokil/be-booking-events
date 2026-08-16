package routes

import (
	authdelivery "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/auth/delivery"
	"github.com/go-chi/chi/v5"
)

func registerDashboard(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// ORGANIZER dan ADMIN: lihat metrik real-time per event
		r.With(authdelivery.RequireRole("ORGANIZER", "ADMIN")).Get("/api/v1/events/{eventID}/metrics", d.Dashboard.GetEventMetrics)
	})
}
