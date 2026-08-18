package routes

import "github.com/go-chi/chi/v5"

func registerAuth(r chi.Router, d Deps) {
	r.Post("/api/v1/auth/register", d.Auth.Register)
	r.Post("/api/v1/auth/login", d.Auth.Login)

	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		r.Get("/api/v1/auth/me", d.Auth.Me)
		r.Post("/api/v1/auth/logout", d.Auth.Logout)

		r.With(d.RequireOrganizer).Post("/api/v1/events/{eventID}/gate-operators", d.Auth.AssignGateOperator)
		r.With(d.RequireOrganizer).Get("/api/v1/events/{eventID}/gate-operators", d.Auth.ListGateOperators)
		r.With(d.RequireOrganizer).Delete("/api/v1/events/{eventID}/gate-operators/{userID}", d.Auth.RemoveGateOperator)
		r.With(d.RequireOrganizer).Get("/api/v1/users", d.Auth.SearchGateOperators)
	})
}
