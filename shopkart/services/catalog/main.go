// ShopKart Catalog Service
// Serves the product catalog. Reads from Postgres (connection details from
// env/Secret). Exposes /products, /products/{id}, /health, /ready.
// The /ready probe verifies the DB is reachable — so the Service only routes
// traffic to pods that can actually serve (the K2/K5 readiness lesson).
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

var db *sql.DB

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func initDB() error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		env("DB_HOST", "postgres"),
		env("DB_PORT", "5432"),
		env("DB_USER", "shopkart"),
		env("DB_PASSWORD", ""), // injected from a Secret
		env("DB_NAME", "catalog"),
		env("DB_SSLMODE", "disable"),
	)
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return nil
}

// ensureSchema creates the table and seeds a few products on first start
// (idempotent — safe to run every boot; real systems use migrations/Jobs).
func ensureSchema() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			price NUMERIC(10,2) NOT NULL,
			stock INT NOT NULL DEFAULT 0
		)`)
	if err != nil {
		return err
	}
	var count int
	db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	if count == 0 {
		_, err = db.Exec(`INSERT INTO products (name, price, stock) VALUES
			('Wireless Mouse', 24.99, 150),
			('Mechanical Keyboard', 89.99, 80),
			('USB-C Hub', 39.99, 200),
			('27" Monitor', 229.99, 35),
			('Laptop Stand', 19.99, 300)`)
	}
	return err
}

func listProducts(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, price, stock FROM products ORDER BY id")
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var p Product
		rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
		products = append(products, p)
	}
	writeJSON(w, http.StatusOK, products)
}

func getProduct(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/products/")
	var p Product
	err := db.QueryRow("SELECT id, name, price, stock FROM products WHERE id=$1", id).
		Scan(&p.ID, &p.Name, &p.Price, &p.Stock)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	if err := ensureSchema(); err != nil {
		log.Printf("schema warning: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	// Readiness verifies the DB connection — pod joins the Service only when DB is reachable.
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db unreachable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("/products/", getProduct)
	mux.HandleFunc("/products", listProducts)

	port := env("PORT", "8080")
	log.Printf("catalog service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
