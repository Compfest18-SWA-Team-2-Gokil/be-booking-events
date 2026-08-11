package routes

import "github.com/go-chi/chi/v5"

func registerAuth(r chi.Router, d Deps) {
	// Public
	r.Post("/api/v1/auth/register", d.Auth.Register)
	r.Post("/api/v1/auth/login", d.Auth.Login)

	// Authenticated
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		r.Get("/api/v1/auth/me", d.Auth.Me)

		// ORGANIZER: assign gate operator ke event
		r.With(d.RequireOrganizer).Post("/api/v1/events/{eventID}/gate-operators", d.Auth.AssignGateOperator)
	})
}
