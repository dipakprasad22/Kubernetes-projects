// ShopKart API Gateway
// Single entry point for the platform. Routes /api/<service>/* to the
// corresponding backend microservice by its Kubernetes Service DNS name.
// Stateless, horizontally scalable (HPA), exposes /health and /ready.
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Backend service base URLs come from env (set via ConfigMap in Kubernetes).
// Defaults use Kubernetes Service DNS names (e.g. http://catalog).
var backends = map[string]string{
	"catalog": env("CATALOG_URL", "http://catalog"),
	"cart":    env("CART_URL", "http://cart"),
	"orders":  env("ORDERS_URL", "http://orders"),
	"users":   env("USERS_URL", "http://users"),
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

// proxy forwards the request to the chosen backend service.
func proxy(w http.ResponseWriter, r *http.Request) {
	// path: /api/<service>/<rest...>
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/"), "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "no service specified", http.StatusBadRequest)
		return
	}
	service := parts[0]
	base, ok := backends[service]
	if !ok {
		http.Error(w, "unknown service: "+service, http.StatusNotFound)
		return
	}
	rest := ""
	if len(parts) == 2 {
		rest = "/" + parts[1]
	}
	targetURL := base + rest
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	// Build the upstream request.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "bad gateway request", http.StatusBadGateway)
		return
	}
	copyHeaders(req.Header, r.Header)

	resp, err := httpClient.Do(req)
	if err != nil {
		// Upstream unreachable / timeout — return 502 (don't crash the gateway).
		log.Printf("upstream error service=%s url=%s err=%v", service, targetURL, err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func main() {
	mux := http.NewServeMux()

	// Liveness: process is up.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Readiness: ready to serve (the gateway has no startup deps, so same as health).
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})
	// All /api/* traffic is proxied to a backend.
	mux.HandleFunc("/api/", proxy)

	port := env("PORT", "8080")
	log.Printf("shopkart gateway listening on :%s, backends=%v", port, backends)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      logging(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// logging is simple request logging middleware (logs to stdout -> container logs).
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
