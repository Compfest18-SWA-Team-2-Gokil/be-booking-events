package routes

import "github.com/go-chi/chi/v5"

func registerInventory(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)
		r.Use(d.QueueGuard)

		// BUYER: hold tiket (dilindungi queue guard saat mode antrian aktif)
		r.With(d.RequireBuyer).Post("/api/v1/tickets/hold", d.Inventory.HoldTicket)
	})
}
