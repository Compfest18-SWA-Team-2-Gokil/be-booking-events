package main

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// @title Booking Events API
// @version 1.0
// @description This is a sample server for Booking Events API.
// @host localhost:8080
// @BasePath /

// RootHandler godoc
// @Summary Get Hello World
// @Description responds with hello world message
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router / [get]
func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "hello world"})
}

// HealthCheckHandler godoc
// @Summary Check API Health
// @Description Check health status and total tickets count in DB
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /check-health [get]
func HealthCheckHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var count int
		err := pool.QueryRow(r.Context(), "SELECT count(*) FROM ticket_units").Scan(&count)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "error",
				"error":  err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "ok",
			"total_tickets": count,
		})
	}
}
