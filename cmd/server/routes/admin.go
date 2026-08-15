package routes

import (
	"github.com/go-chi/chi/v5"
)

func registerAdmin(r chi.Router, d Deps) {
	if d.Admin == nil {
		return
	}
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.Use(d.AuthMiddleware)
		r.Use(d.RequireAdmin)

		r.Get("/disputes", d.Admin.ListDisputes)
		r.Post("/orders/{orderID}/override", d.Admin.OverrideOrderStatus)
		r.Post("/tickets/{unitID}/reassign", d.Admin.ReassignTicket)
		r.Get("/audit-logs", d.Admin.ListAuditLogs)
	})
}
