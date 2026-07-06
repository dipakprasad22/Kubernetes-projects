# RatingsBoard — Media-Measurement Reporting & Dashboards on Kubernetes

A Nielsen-style media-measurement **reporting platform** on Kubernetes. Built **local-first on KIND**, then deployed to **Amazon EKS**, packaged with **Helm**. It consumes **PanelPulse (P3)**'s `measurement_aggregates`, computes ratings and share, and serves them through a monitored API and dashboard — closing the loop into an **end-to-end measurement system** (ingest → process → report).

> Part of the [kubernetes-portfolio](../) — Project 4 of 4, the finale. Leans into **observability & operations**.
>
> **Original, generic reference design** using public measurement concepts (ratings, share, impressions, reach). Not based on any proprietary system.

---

## What this project demonstrates

| Kubernetes concept | Where it shows up in RatingsBoard |
|---|---|
| **Prometheus metrics** | The reporting API exposes `/metrics` (request rate, errors, latency — the RED method) |
| **Prometheus + Grafana** | kube-prometheus-stack scrapes the API; Grafana dashboards on RED; sample alert rule |
| **ServiceMonitor / PrometheusRule** | Operator-native scrape config + a symptom-based alert (error rate > 5%) |
| **CronJob** | Scheduled rating calculation reads aggregates → writes the reports table |
| **ResourceQuota + LimitRange** | Per-namespace caps + per-pod defaults — the K6 multi-tenancy operations material |
| **Helm** | The whole platform packaged with per-environment values |
| **Deployment + HPA** | Reporting API scales on CPU; zero-downtime rolling deploys |
| **StatefulSet** | Reports Postgres (local); RDS on EKS |
| **Ingress** | `/` → dashboard, `/api` → reporting API |
| **securityContext / Pod Security** | Non-root, drop caps, read-only FS; namespace `baseline`/`restricted` |

---

## The end-to-end system (P3 + P4)

```
[ P3 PanelPulse ]  meters → collectors → Kafka → processors → aggregates
                                                                   │
                                                                   ▼  (measurement_aggregates)
[ P4 RatingsBoard ]  rating-calc CronJob → reports DB → Reporting API → Dashboard
                                                              │
                                                         /metrics → Prometheus → Grafana
```

RatingsBoard reads the aggregates PanelPulse produces, so the two projects together tell one story: **how raw measurement events become published ratings**, the full lifecycle a real measurement company runs.

---

## Observability (the lean)

The reporting API is instrumented for **Prometheus** using the **RED method**:
- **Rate** — `http_requests_total` (requests/sec)
- **Errors** — the 5xx slice of that counter
- **Duration** — `http_request_duration_seconds` histogram (p95 latency)

Prometheus scrapes `/metrics` (via the `ServiceMonitor`), Grafana dashboards it, and a `PrometheusRule` alerts on the **symptom** (sustained error rate > 5%) rather than a cause. See [`monitoring/`](monitoring/) for install steps, the ServiceMonitor, the alert, and the PromQL panels. This is the K6 observability stack made real on an app you can actually watch.

---

## Operations (the lean)

- **Scheduled batch:** the rating-calc **CronJob** runs hourly (`concurrencyPolicy: Forbid`), computing ratings from the latest aggregates.
- **Multi-tenancy:** a **ResourceQuota** caps the namespace's total CPU/memory/pods and a **LimitRange** gives unset pods sane defaults — so this tenant shares the cluster fairly (the K6 namespace operations material).
- **Helm:** one chart, per-environment values, versioned/rollback-able releases.

---

## Services

| Service | Language | Role | Workload |
|---|---|---|---|
| `reporting-api` | Python / FastAPI | Serve ratings + Prometheus `/metrics` | Deployment + HPA |
| `rating-calc` | Python | Compute ratings & share from aggregates | CronJob |
| `dashboard` | Node.js | Visualize ratings | Deployment |
| `reports-db` | Postgres | Reports store | StatefulSet / RDS |

**Ratings model:** `rating = reach / panel_universe × 100`; `share = impressions / daypart_total_impressions × 100`.

---

## Quick start (KIND)

```bash
kind create cluster --name ratingsboard --config manifests/kind/kind-cluster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
./manifests/kind/build-and-load.sh ratingsboard
kubectl apply -f manifests/base/
kubectl -n ratingsboard get pods,svc,hpa,cronjob,resourcequota

# compute ratings on demand (seeds the reports table; reads PanelPulse aggregates if present)
kubectl -n ratingsboard create job --from=cronjob/rating-calc calc-now
kubectl -n ratingsboard logs job/calc-now
open http://localhost/            # the dashboard
curl http://localhost/api/ratings/top
```

Monitoring, full KIND walkthrough, and EKS guide: [`monitoring/README.md`](monitoring/README.md), [`docs/kind-guide.md`](docs/kind-guide.md), [`docs/eks-guide.md`](docs/eks-guide.md).

---

## Production notes

- **On EKS:** reports DB → **RDS**; observability → **Amazon Managed Prometheus + Grafana** (or self-managed stack); Ingress → **ALB**. Source aggregates come from PanelPulse's store or a shared warehouse.
- **Alert on symptoms, not causes:** the included alert watches user-facing error rate — the RED/SLO discipline.
- **Quota tuning:** the ResourceQuota is a starting point — size it from real usage; combine with RBAC + NetworkPolicy for full tenant isolation.
