package routes

import "github.com/go-chi/chi/v5"

func registerQueue(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// BUYER: masuk dan cek status antrean
		r.With(d.RequireBuyer).Post("/api/v1/events/{eventID}/queue/join", d.Queue.JoinQueue)
		r.With(d.RequireBuyer).Get("/api/v1/events/{eventID}/queue/status", d.Queue.GetQueueStatus)

		// Any authenticated: validasi queue token sebelum checkout
		r.Post("/api/v1/queue/token/validate", d.Queue.ValidateToken)
	})
}
