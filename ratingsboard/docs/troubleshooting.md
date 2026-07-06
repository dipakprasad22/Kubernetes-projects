# RatingsBoard Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Dashboard shows "No ratings yet" | rating-calc hasn't run / no source aggregates | run the job; ensure PanelPulse aggregates exist (run P3) |
| reporting-api readiness failing | reports DB unreachable | check reports-db StatefulSet Ready; verify DB_PASSWORD |
| rating-calc Job exits 0 but no rows | source aggregates empty/unreachable | it degrades gracefully; confirm SRC_DB_* points at PanelPulse data |
| Prometheus not scraping API | ServiceMonitor label mismatch | the `release: kube-prom` label must match your stack's serviceMonitorSelector |
| Grafana "No data" | wrong datasource / PromQL / time range | verify Prometheus datasource + the RED queries |
| Pods rejected: "exceeded quota" | ResourceQuota cap hit | `kubectl describe resourcequota`; raise quota or reduce usage |
| Pods rejected: "must specify limits" | quota requires them, none set | the LimitRange supplies defaults — ensure it's applied |
| HPA `<unknown>` | metrics-server missing | install with `--kubelet-insecure-tls` on KIND |
| Alert never fires | no error traffic / rule label mismatch | generate 5xx; check PrometheusRule `release` label |

**Mental model:** aggregates (from P3) → rating-calc (CronJob) → reports DB → reporting-api (/metrics) → dashboard,
with Prometheus/Grafana observing and ResourceQuota/LimitRange bounding. Walk the chain to localize a problem.
