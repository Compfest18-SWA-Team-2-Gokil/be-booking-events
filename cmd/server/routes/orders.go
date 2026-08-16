package routes

import (
	ordersdelivery "github.com/Compfest18-SWA-Team-2-Gokil/be-booking-events/internal/orders/delivery"
	"github.com/go-chi/chi/v5"
)

func registerOrders(r chi.Router, d Deps) {
	// Webhook Xendit — public, tidak pakai JWT
	r.Post("/api/v1/payments/webhook/xendit", d.Orders.XenditWebhook)

	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// BUYER: buat order dilindungi queue guard + idempotency
		r.With(d.RequireBuyer, d.QueueGuard, d.Idempotency).Post("/api/v1/orders", d.Orders.CreateOrder)
		// /orders/my harus sebelum /orders/{orderID} supaya chi tidak menganggap "my" sebagai orderID param
		r.With(d.RequireBuyer).Get("/api/v1/orders/my", d.Orders.GetMyOrders)
		r.With(d.RequireBuyer).Get("/api/v1/orders/{orderID}", d.Orders.GetOrder)
		r.With(d.RequireBuyer, d.Idempotency).Post("/api/v1/orders/{orderID}/pay", d.Orders.InitiatePayment)
		r.With(d.RequireBuyer).Post("/api/v1/orders/{orderID}/refund", d.Orders.RequestRefund)

		// ORGANIZER: approve refund & lihat daftar pengajuan refund
		r.With(d.RequireOrganizer).Get("/api/v1/orders/organizer/refunds", d.Orders.ListOrganizerRefunds)
		r.With(d.RequireOrganizer).Post("/api/v1/orders/{orderID}/refund/approve", d.Orders.ApproveRefund)
	})
}

// pastikan compiler tahu tipe Orders di Deps
var _ *ordersdelivery.OrdersHandler = (*ordersdelivery.OrdersHandler)(nil)
