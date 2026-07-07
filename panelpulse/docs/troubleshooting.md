# PanelPulse Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| Collector returns 5xx on /ingest | Kafka unreachable | check Kafka StatefulSet Ready; verify KAFKA_BOOTSTRAP; collector should still 202 if buffering to log-stub |
| Processors not consuming | wrong topic/consumer group / Kafka down | check CONSUMER_GROUP + KAFKA_TOPIC; `kubectl logs -l app=processor` |
| Consumer lag growing unbounded | too few processors / slow DB | scale processors (or KEDA); check results-db performance |
| DaemonSet pod missing on a node | node tainted | the DaemonSet tolerates all taints; check node status |
| StatefulSet pod Pending | PVC unbound (no storage class) | ensure a default StorageClass (KIND has one; EKS needs EBS CSI add-on) |
| Aggregator Job fails | DB unreachable / empty source | check DB_PASSWORD secret + results-db Ready; verify processed_events has rows |
| HPA `<unknown>` | metrics-server missing | install with `--kubelet-insecure-tls` on KIND |
| Lag-based scaling not working | KEDA not installed / wrong scaler config | install KEDA; verify ScaledObject triggers point at the right brokers/topic |
| EKS: can't reach MSK | SG / subnet | MSK brokers and nodes must share VPC/SG access on 9092 |

**Pipeline mental model:** ingest (collector) → buffer (Kafka) → process (processor) → store (DB) → aggregate (CronJob).
Diagnose by walking the stages: is ingest accepting (202)? is Kafka receiving? are processors consuming (lag)? is the DB filling? did aggregation run?
