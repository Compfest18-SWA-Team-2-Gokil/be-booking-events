package infrastructure

import (
	"context"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/dashboard/application"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresMetricsRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresMetricsRepository(pool *pgxpool.Pool) *PostgresMetricsRepository {
	return &PostgresMetricsRepository{pool: pool}
}

var _ application.MetricsRepository = (*PostgresMetricsRepository)(nil)

// GetEventMetrics menjalankan query agregasi langsung ke tabel ticket_units (source of truth).
// Index idx_ticket_units_type_status memastikan ini tidak menjadi full table scan.
func (r *PostgresMetricsRepository) GetEventMetrics(ctx context.Context, eventID string) ([]application.TicketTypeMetrics, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			ticket_type_id,
			status,
			COUNT(*) AS jumlah
		FROM ticket_units
		WHERE ticket_type_id IN (
			SELECT id FROM ticket_types WHERE event_id = $1
		)
		GROUP BY ticket_type_id, status
		ORDER BY ticket_type_id
	`, eventID)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	// Kumpulkan per ticket_type_id, agregasi status di Go.
	type key = string
	metricsMap := make(map[key]*application.TicketTypeMetrics)
	var order []key

	for rows.Next() {
		var typeID, status string
		var count int
		if err := rows.Scan(&typeID, &status, &count); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		m, ok := metricsMap[typeID]
		if !ok {
			m = &application.TicketTypeMetrics{TicketTypeID: typeID}
			metricsMap[typeID] = m
			order = append(order, typeID)
		}

		switch status {
		case "AVAILABLE":
			m.Available = count
		case "HELD":
			m.Held = count
		case "PAYMENT_PENDING", "CONFIRMED":
			m.Sold += count
		case "ADMITTED":
			m.Admitted = count
		case "REFUNDED":
			m.Refunded = count
		}
		m.Total += count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	result := make([]application.TicketTypeMetrics, 0, len(order))
	for _, id := range order {
		result = append(result, *metricsMap[id])
	}
	return result, nil
}
