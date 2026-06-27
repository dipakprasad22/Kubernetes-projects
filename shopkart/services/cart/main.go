// ShopKart Cart Service — stores shopping carts in Redis (fast session state).
// Exposes /cart/{userID} (GET/POST/DELETE), /health, /ready.
// /ready verifies Redis connectivity.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx = context.Background()

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type CartItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     env("REDIS_HOST", "redis") + ":" + env("REDIS_PORT", "6379"),
		Password: env("REDIS_PASSWORD", ""),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		c, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := rdb.Ping(c).Err(); err != nil {
			http.Error(w, `{"status":"redis unreachable"}`, http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"status":"ready"}`))
	})
	mux.HandleFunc("/cart/", handleCart)

	port := env("PORT", "8080")
	log.Printf("cart service listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleCart(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/cart/")
	if userID == "" {
		http.Error(w, "user id required", http.StatusBadRequest)
		return
	}
	key := "cart:" + userID
	switch r.Method {
	case http.MethodGet:
		val, err := rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"items":[]}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(val))
	case http.MethodPost:
		var items []CartItem
		json.NewDecoder(r.Body).Decode(&items)
		b, _ := json.Marshal(map[string]interface{}{"items": items})
		rdb.Set(ctx, key, b, 24*time.Hour) // carts expire after 24h
		w.WriteHeader(http.StatusCreated)
		w.Write(b)
	case http.MethodDelete:
		rdb.Del(ctx, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
