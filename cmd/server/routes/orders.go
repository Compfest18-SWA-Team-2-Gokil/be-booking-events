package routes

import (
	ordersdelivery "github.com/ebk-tech/be-booking-events/internal/orders/delivery"
	"github.com/go-chi/chi/v5"
)

func registerOrders(r chi.Router, d Deps) {
	// Webhook Xendit — public, tidak pakai JWT
	r.Post("/api/v1/payments/webhook/xendit", d.Orders.XenditWebhook)

	r.Group(func(r chi.Router) {
		r.Use(d.AuthMiddleware)

		// BUYER: buat order + inisiasi bayar dilindungi idempotency key & queue token
		r.With(d.RequireBuyer, d.RequireQueueToken, d.Idempotency).Post("/api/v1/orders", d.Orders.CreateOrder)
		r.With(d.RequireBuyer).Get("/api/v1/orders/{orderID}", d.Orders.GetOrder)
		r.With(d.RequireBuyer, d.Idempotency).Post("/api/v1/orders/{orderID}/pay", d.Orders.InitiatePayment)
		r.With(d.RequireBuyer).Post("/api/v1/orders/{orderID}/refund", d.Orders.RequestRefund)

		// ORGANIZER: approve refund
		r.With(d.RequireOrganizer).Post("/api/v1/orders/{orderID}/refund/approve", d.Orders.ApproveRefund)

		// Respon Webhook
		
	})
}

// pastikan compiler tahu tipe Orders di Deps
var _ *ordersdelivery.OrdersHandler = (*ordersdelivery.OrdersHandler)(nil)
