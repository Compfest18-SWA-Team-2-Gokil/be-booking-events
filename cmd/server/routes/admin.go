package routes

import "github.com/go-chi/chi/v5"

func registerAdmin(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)
		r.Use(d.RequireAdmin)

		r.Get("/api/v1/admin/disputes", d.Admin.ListDisputes)
		r.Post("/api/v1/admin/orders/{orderID}/override", d.Admin.OverrideOrder)
		r.Post("/api/v1/admin/tickets/{unitID}/reassign", d.Admin.ReassignTicket)
		r.Get("/api/v1/admin/audit-logs", d.Admin.ListAuditLogs)
	})
}
