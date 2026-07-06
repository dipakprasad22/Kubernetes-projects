# RatingsBoard Observability — Prometheus + Grafana

The reporting API exposes a Prometheus `/metrics` endpoint (request rate, errors,
latency — the RED method). This directory shows how to monitor it.

## Install the kube-prometheus-stack (Helm)
```bash
kubectl create namespace monitoring
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prom prometheus-community/kube-prometheus-stack -n monitoring
```

## Scrape the reporting API
The reporting-api pods carry `prometheus.io/scrape` annotations. With the
prometheus-stack, add a ServiceMonitor (or use annotation-based scraping) so
Prometheus picks up `reporting-api:8000/metrics`. A ServiceMonitor is provided:
`servicemonitor.yaml`.

## Access Grafana
```bash
kubectl -n monitoring port-forward svc/kube-prom-grafana 3000:80
# user admin / get password:
kubectl -n monitoring get secret kube-prom-grafana -o jsonpath="{.data.admin-password}" | base64 -d
```

## RED dashboard panels (PromQL)
- **Rate:**     `sum(rate(http_requests_total{path=~"/api/.*"}[5m]))`
- **Errors:**   `sum(rate(http_requests_total{status=~"5.."}[5m]))`
- **Duration:** `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le, path))`

## Sample alert
`alert.yaml` fires when the API error rate exceeds 5% for 5 minutes — alert on
the user-facing SYMPTOM (errors), not a cause.
