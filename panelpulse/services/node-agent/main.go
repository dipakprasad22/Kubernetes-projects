// PanelPulse Node Agent — runs as a DaemonSet (one per node). Collects
// node-level metrics relevant to the ingest pipeline (host load, local event
// counts) and exposes them on /metrics for Prometheus scraping. Demonstrates
// the "one pod per node" agent pattern (like Fluent Bit / node-exporter).
package main

import (
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	node := env("NODE_NAME", "unknown")
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		// Minimal node metrics; real agents read /proc, cgroups, etc.
		w.Write([]byte(
			"panelpulse_node_goroutines{node=\"" + node + "\"} " + itoa(runtime.NumGoroutine()) + "\n" +
				"panelpulse_node_up{node=\"" + node + "\"} 1\n"))
	})
	port := env("PORT", "9100")
	log.Printf("panelpulse node-agent on :%s node=%s", port, node)
	go func() {
		for {
			time.Sleep(30 * time.Second)
			log.Printf("node-agent heartbeat node=%s", node)
		}
	}()
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
