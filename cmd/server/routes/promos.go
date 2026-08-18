package routes

import (
	"github.com/go-chi/chi/v5"
)

func registerPromos(r chi.Router, d Deps) {
	if d.Promos == nil {
		return
	}

	// Public routes
	r.Route("/api/v1/promos", func(r chi.Router) {
		r.Post("/validate", d.Promos.ValidatePromo)
		r.Get("/active", d.Promos.ListActivePromos)
	})

	// Admin routes
	r.Route("/api/v1/admin/promos", func(r chi.Router) {
		r.Use(d.AuthMiddleware)
		r.Use(d.RequireAdmin)

		r.Get("/", d.Promos.AdminListPromos)
		r.Post("/", d.Promos.AdminCreatePromo)
		r.Put("/{id}", d.Promos.AdminUpdatePromo)
		r.Delete("/{id}", d.Promos.AdminDeletePromo)
	})
}
