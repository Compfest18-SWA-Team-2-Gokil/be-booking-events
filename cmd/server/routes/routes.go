package routes

import (
	"net/http"

	admindelivery "github.com/ebk-tech/be-booking-events/internal/admin/delivery"
	authdelivery "github.com/ebk-tech/be-booking-events/internal/auth/delivery"
	checkindelivery "github.com/ebk-tech/be-booking-events/internal/checkin/delivery"
	dashboarddelivery "github.com/ebk-tech/be-booking-events/internal/dashboard/delivery"
	eventsdelivery "github.com/ebk-tech/be-booking-events/internal/events/delivery"
	inventorydelivery "github.com/ebk-tech/be-booking-events/internal/inventory/delivery"
	ordersdelivery "github.com/ebk-tech/be-booking-events/internal/orders/delivery"
	queuedelivery "github.com/ebk-tech/be-booking-events/internal/queue/delivery"
	"github.com/go-chi/chi/v5"
)

// Deps menampung semua middleware dan handler yang dibutuhkan oleh router.
// Diisi di main.go setelah semua dependency di-wire.
type Deps struct {
	// Middleware
	AuthMiddleware      func(http.Handler) http.Handler
	RequireBuyer        func(http.Handler) http.Handler
	RequireOrganizer    func(http.Handler) http.Handler
	RequireGateOperator func(http.Handler) http.Handler
	RequireAdmin        func(http.Handler) http.Handler
	Idempotency         func(http.Handler) http.Handler
	QueueGuard          func(http.Handler) http.Handler

	// Handlers
	Auth      *authdelivery.AuthHandler
	Admin     *admindelivery.AdminHandler
	Inventory *inventorydelivery.InventoryHandler
	Dashboard *dashboarddelivery.DashboardHandler
	Checkin   *checkindelivery.CheckinHandler
	Queue     *queuedelivery.QueueHandler
	Events    *eventsdelivery.EventsHandler
	Orders    *ordersdelivery.OrdersHandler
}

// Register mendaftarkan semua route ke router.
func Register(r chi.Router, d Deps) {
	registerDocs(r)
	registerAuth(r, d)
	registerAdmin(r, d)
	registerEvents(r, d)
	registerInventory(r, d)
	registerDashboard(r, d)
	registerCheckin(r, d)
	registerQueue(r, d)
	registerOrders(r, d)
	registerDebug(r, d)
}
