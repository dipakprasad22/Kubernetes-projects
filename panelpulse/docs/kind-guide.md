# PanelPulse on KIND — Full Walkthrough

## Prerequisites
Docker, kind, kubectl, (optional) helm. The 2-worker config lets the DaemonSet
show one node-agent per worker. Maven build runs inside the Docker build.

## 1. Cluster + ingress + metrics-server
```bash
kind create cluster --name panelpulse --config manifests/kind/kind-cluster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
```

## 2. Build + deploy
```bash
./manifests/kind/build-and-load.sh panelpulse
kubectl apply -f manifests/base/
kubectl -n panelpulse get pods -w   # Kafka + DB (StatefulSet) come up first, then the rest
```

## 3. Send events and watch the pipeline
```bash
# ingest an exposure event (returns 202 Accepted immediately)
curl -i -X POST http://localhost/ingest -H 'Content-Type: application/json' \
  -d '{"panelist_id":"p-1001","device_id":"meter-7","channel_id":"ch-42","started_at":"2026-01-01T20:00:00Z","duration_sec":1800}'

# collector metrics (accepted/rejected counters)
kubectl -n panelpulse exec deploy/collector -- wget -qO- localhost:8080/metrics

# processors consuming from Kafka
kubectl -n panelpulse logs -l app=processor --tail=20
```

## 4. See the workload types in action
```bash
# DaemonSet: one node-agent per node
kubectl -n panelpulse get pods -l app=node-agent -o wide      # 2 pods on 2 workers

# StatefulSet: stable pod names + PVCs
kubectl -n panelpulse get statefulset,pvc

# HPA on collectors (generate load to watch it scale)
kubectl -n panelpulse get hpa -w
```

## 5. Aggregation (the batch roll-up)
```bash
kubectl -n panelpulse create job --from=cronjob/aggregator agg-now
kubectl -n panelpulse logs job/agg-now
# rolls processed_events -> measurement_aggregates (impressions, viewing minutes, reach)
kubectl -n panelpulse exec statefulset/results-db -- \
  psql -U panelpulse -d panelpulse -c "SELECT * FROM measurement_aggregates LIMIT 10;"
```

## 6. Backpressure demo (the point of the buffer)
```bash
# Scale processors to 0, flood ingest — events queue in Kafka, nothing is lost:
kubectl -n panelpulse scale deploy/processor --replicas=0
# (send many events) ... then scale processors back up and watch them drain the lag:
kubectl -n panelpulse scale deploy/processor --replicas=3
kubectl -n panelpulse logs -l app=processor --tail=30
```

## Teardown
```bash
kind delete cluster --name panelpulse
```
