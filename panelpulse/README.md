# PanelPulse — Media-Measurement Ingest & Processing Pipeline on Kubernetes

A Nielsen-style media-measurement **ingest and processing pipeline** on Kubernetes. Built **local-first on KIND**, then deployed to **Amazon EKS**, packaged with **Helm**. Where ShopKart proved microservices and FinLedger proved security, PanelPulse proves **data-pipeline engineering at scale** — high-volume ingest, durable buffering, lag-aware stream processing, and batch aggregation.

> Part of the [kubernetes-portfolio](../) — Project 3 of 4. Pairs with **P4 RatingsBoard** (reporting) to form an end-to-end measurement system.
>
> **Original, generic reference design** using public measurement concepts (panels, meters, exposure events, impressions, viewing minutes, reach). Not based on any proprietary system.

---

## What this project demonstrates

| Kubernetes concept | Where it shows up in PanelPulse |
|---|---|
| **StatefulSet** | Kafka (durable buffer) and the results Postgres DB — stable identity + per-pod PVC |
| **DaemonSet** | `node-agent` — exactly one per node for node-level health/metrics |
| **Deployment + HPA** | Collectors (scale on CPU/ingest) and processors (scale on consumer lag) |
| **CronJob** | Aggregation batch — rolls up processed events into measurement aggregates |
| **Jobs (run-to-completion)** | The aggregator runs and exits; `concurrencyPolicy: Forbid` prevents overlap |
| **PV / PVC** | Kafka and DB persistence via `volumeClaimTemplates` (EBS on EKS) |
| **Resource tuning** | Lightweight Go collectors (64Mi) vs heavier JVM processors (512Mi–1Gi) |
| **Ingress** | Meters POST exposure events to `/ingest` |
| **securityContext** | Non-root, drop caps, read-only FS across all workloads |

---

## The pipeline

```
Panel meters ──POST /ingest──▶ Collectors (Go)        [Deployment + HPA, scale on ingest]
                                    │ produce
                                    ▼
                              Kafka (StatefulSet)       [durable BUFFER — absorbs spikes]
                                    │ consume
                                    ▼
                              Processors (Java)         [Deployment + HPA on CONSUMER LAG]
                                    │ validate · enrich (daypart) · dedup
                                    ▼
                              Results DB (Postgres)     [processed_events]
                                    │ read
                                    ▼
                              Aggregation CronJob       [→ measurement_aggregates]
                                    │
                                    ▼
                         impressions · viewing minutes · reach   (consumed by P4 RatingsBoard)

node-agent (DaemonSet) ── one per node ── node-level health/metrics
```

**The key idea — the buffer.** Kafka sits between bursty ingest and steady processing. When meters spike (prime-time, a big live event), events queue durably in Kafka instead of overwhelming the processors — that's **backpressure handling**. Collectors accept fast (return `202 Accepted`) and never block on downstream work; processors drain the queue at their own pace and **scale on consumer lag** (how far behind they are), which is the correct scaling signal for a stream pipeline.

---

## Services

| Service | Language | Role | Workload type |
|---|---|---|---|
| `collector` | Go | High-volume ingest → produce to Kafka | Deployment + HPA |
| `processor` | Java / Spring | Consume, validate, enrich (daypart), dedup → DB | Deployment + HPA (lag) |
| `aggregator` | Java | Roll up processed events → aggregates | CronJob (run-to-completion) |
| `node-agent` | Go | Node-level health/metrics | DaemonSet |
| `kafka` | (image) | Durable event buffer | StatefulSet |
| `results-db` | Postgres | Processed events + aggregates | StatefulSet / RDS |

---

## The measurement model (generic, public concepts)

- **Exposure event:** a panelist's device recorded exposure to a channel for some seconds.
- **Processed event:** validated, deduped, enriched with **daypart** (e.g. prime-time).
- **Aggregates** (by channel + daypart): **impressions** = event count, **viewing minutes** = Σ duration / 60, **reach** = distinct panelists.

---

## Quick start (KIND)

```bash
kind create cluster --name panelpulse --config manifests/kind/kind-cluster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
./manifests/kind/build-and-load.sh panelpulse
kubectl apply -f manifests/base/
kubectl -n panelpulse get pods,statefulset,daemonset,hpa,cronjob

# Send an exposure event:
curl -X POST http://localhost/ingest -H 'Content-Type: application/json' \
  -d '{"panelist_id":"p-1001","device_id":"meter-7","channel_id":"ch-42","started_at":"2026-01-01T20:00:00Z","duration_sec":1800}'
# -> 202 Accepted

# Watch the DaemonSet place one node-agent per node:
kubectl -n panelpulse get pods -l app=node-agent -o wide

# Run aggregation on demand:
kubectl -n panelpulse create job --from=cronjob/aggregator agg-now
kubectl -n panelpulse logs job/agg-now
```

Full walkthrough: [`docs/kind-guide.md`](docs/kind-guide.md). EKS (MSK + RDS + KEDA): [`docs/eks-guide.md`](docs/eks-guide.md).

---

## Production notes

- **On EKS:** Kafka → **Amazon MSK**, results DB → **RDS**, lag-based scaling → **KEDA** (Kafka scaler). Keep the cluster stateless; let AWS run the durable systems. See `manifests/eks/README-eks-datapipeline.md`.
- **Scaling signal:** CPU HPA is the portable default here; the *right* signal for processors is **consumer lag** (KEDA `ScaledObject`, pattern included in `manifests/base/11-processor.yaml`).
- **Backpressure:** the collector returns `202` and never blocks — Kafka is the shock absorber. Size broker storage for your worst-case spike duration.
- **Resource tuning:** Go collectors are deliberately tiny (high concurrency, low overhead); JVM processors get more memory. This contrast is intentional and realistic.
