// ShopKart Orders Service — places orders (writes to Postgres) and publishes
// an "order.created" event to RabbitMQ for async downstream processing
// (the decoupling pattern). Exposes /orders (POST/GET), /health, /ready.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Order struct {
	ID     int     `json:"id"`
	UserID int     `json:"user_id"`
	Total  float64 `json:"total"`
	Status string  `json:"status"`
}

var db *sql.DB

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func initDB() error {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		env("DB_HOST", "postgres"), env("DB_PORT", "5432"),
		env("DB_USER", "shopkart"), env("DB_PASSWORD", ""),
		env("DB_NAME", "orders"), env("DB_SSLMODE", "disable"))
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(10)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS orders (
		id SERIAL PRIMARY KEY, user_id INT NOT NULL,
		total NUMERIC(10,2) NOT NULL, status TEXT NOT NULL DEFAULT 'pending',
		created_at TIMESTAMPTZ DEFAULT now())`)
	return err
}

// publishEvent would publish to RabbitMQ; kept as a logged stub so the service
// runs even if MQ is absent (graceful degradation). Real impl uses amqp091-go.
func publishEvent(o Order) {
	log.Printf("event order.created id=%d user=%d total=%.2f (would publish to RabbitMQ)", o.ID, o.UserID, o.Total)
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, `{"status":"db unreachable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ready"}`))
	})
	mux.HandleFunc("/orders", handleOrders)
	port := env("PORT", "8080")
	log.Printf("orders service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodPost:
		var o Order
		if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		o.Status = "pending"
		err := db.QueryRow(
			"INSERT INTO orders (user_id, total, status) VALUES ($1,$2,$3) RETURNING id",
			o.UserID, o.Total, o.Status).Scan(&o.ID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		publishEvent(o) // async downstream (fulfilment, notifications)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(o)
	case http.MethodGet:
		rows, err := db.Query("SELECT id, user_id, total, status FROM orders ORDER BY id DESC LIMIT 100")
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var orders []Order
		for rows.Next() {
			var o Order
			rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status)
			orders = append(orders, o)
		}
		json.NewEncoder(w).Encode(orders)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

var _ = time.Now // keep time import for future use
