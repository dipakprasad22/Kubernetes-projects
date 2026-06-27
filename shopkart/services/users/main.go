// ShopKart Users Service — account registration and lookup (Postgres).
// Passwords hashed with bcrypt. Exposes /register, /users/{id}, /health, /ready.
package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type registerReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
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
		env("DB_NAME", "users"), env("DB_SSLMODE", "disable"))
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY, email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL, password_hash TEXT NOT NULL)`)
	return err
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("db init failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"status":"ok"}`)) })
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, `{"status":"db unreachable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ready"}`))
	})
	mux.HandleFunc("/register", register)
	mux.HandleFunc("/users/", getUser)
	port := env("PORT", "8080")
	log.Printf("users service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	var u User
	err := db.QueryRow(
		"INSERT INTO users (email, name, password_hash) VALUES ($1,$2,$3) RETURNING id, email, name",
		req.Email, req.Name, string(hash)).Scan(&u.ID, &u.Email, &u.Name)
	if err != nil {
		http.Error(w, "email already exists or db error", http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(u)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/users/")
	var u User
	err := db.QueryRow("SELECT id, email, name FROM users WHERE id=$1", id).Scan(&u.ID, &u.Email, &u.Name)
	if err == sql.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(u)
}
