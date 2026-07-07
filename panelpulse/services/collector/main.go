// PanelPulse Ingest Collector
//
// High-volume ingest endpoint for media-measurement events. Panel devices and
// meters POST exposure events (a panelist watched a channel for some seconds);
// the collector validates lightly and ENQUEUES them to Kafka as fast as
// possible, so ingest is decoupled from (slower) downstream processing.
//
// Design goals:
//   - Accept fast, never block on processing — Kafka absorbs spikes (backpressure).
//   - Stateless & horizontally scalable (HPA) — add collectors under load.
//   - Lightweight (Go) for high concurrency and low per-request overhead.
//
// This is an ORIGINAL, generic reference design using public measurement
// concepts (panels, meters, impressions). It is not based on any proprietary system.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

// ExposureEvent is a single media-exposure measurement record.
type ExposureEvent struct {
	PanelistID  string    `json:"panelist_id"`  // anonymized panel member id
	DeviceID    string    `json:"device_id"`    // meter / device id
	ChannelID   string    `json:"channel_id"`   // media/channel identifier
	StartedAt   time.Time `json:"started_at"`   // when exposure began
	DurationSec int       `json:"duration_sec"` // seconds of exposure
	ReceivedAt  time.Time `json:"received_at"`  // server receive time (set here)
}

var (
	accepted int64 // counter for /metrics
	rejected int64
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// producer is the interface to Kafka. We keep it abstract so the collector can
// run with a real Kafka producer or a logging stub (graceful in dev/KIND).
type producer interface {
	Produce(ctx context.Context, topic string, key string, value []byte) error
	Close() error
}

// logProducer is a stub that logs instead of producing — lets the service run
// end-to-end without Kafka for local smoke tests. Swap for kafkaProducer in prod.
type logProducer struct{}

func (l *logProducer) Produce(ctx context.Context, topic, key string, value []byte) error {
	log.Printf("produce topic=%s key=%s bytes=%d", topic, key, len(value))
	return nil
}
func (l *logProducer) Close() error { return nil }

var prod producer = &logProducer{}
var topic = env("KAFKA_TOPIC", "exposure-events")

func ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var ev ExposureEvent
	if err := json.NewDecoder(r.Body).Decode(&ev); err != nil {
		atomic.AddInt64(&rejected, 1)
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}
	// Light validation — keep the hot path cheap; deep validation is downstream.
	if ev.PanelistID == "" || ev.ChannelID == "" || ev.DurationSec <= 0 {
		atomic.AddInt64(&rejected, 1)
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}
	ev.ReceivedAt = time.Now().UTC()

	value, _ := json.Marshal(ev)
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	// Key by panelist for partition affinity (ordering per panelist).
	if err := prod.Produce(ctx, topic, ev.PanelistID, value); err != nil {
		atomic.AddInt64(&rejected, 1)
		// Enqueue failed — tell the client to retry (don't silently drop).
		http.Error(w, "enqueue failed", http.StatusServiceUnavailable)
		return
	}
	atomic.AddInt64(&accepted, 1)
	w.WriteHeader(http.StatusAccepted) // 202: accepted for async processing
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	// Readiness could verify the Kafka connection in a real producer.
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ready"}`))
	})
	// Prometheus-style metrics (accepted/rejected counters) for monitoring + HPA basis.
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		log.Printf("metrics accepted=%d rejected=%d", atomic.LoadInt64(&accepted), atomic.LoadInt64(&rejected))
		w.Write([]byte(
			"panelpulse_events_accepted_total " + itoa(atomic.LoadInt64(&accepted)) + "\n" +
				"panelpulse_events_rejected_total " + itoa(atomic.LoadInt64(&rejected)) + "\n"))
	})
	mux.HandleFunc("/ingest", ingest)

	port := env("PORT", "8080")
	log.Printf("panelpulse collector listening on :%s topic=%s", port, topic)
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

// itoa avoids importing strconv just for one use.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
