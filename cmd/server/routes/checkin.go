package routes

import "github.com/go-chi/chi/v5"

func registerCheckin(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// ORGANIZER: generate QR code untuk tiket CONFIRMED
		r.With(d.RequireOrganizer).Post("/api/v1/checkin/issue", d.Checkin.IssueTicket)

		// GATE_OPERATOR: scan QR untuk admit (validasi assignment otomatis)
		r.With(d.RequireGateOperator).Post("/api/v1/checkin/scan", d.Checkin.ScanTicket)
	})
}
