package routes

import "github.com/go-chi/chi/v5"

func registerCheckin(r chi.Router, d Deps) {
	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// BUYER (dan ORGANIZER): generate/tampilkan QR code tiket milik sendiri.
		// Validasi kepemilikan tiket dilakukan di layer use case (GetConfirmedUnit).
		r.Post("/api/v1/checkin/issue", d.Checkin.IssueTicket)

		// GATE_OPERATOR: scan QR untuk admit (validasi assignment otomatis di use case)
		r.With(d.RequireGateOperator).Post("/api/v1/checkin/scan", d.Checkin.ScanTicket)
	})
}
