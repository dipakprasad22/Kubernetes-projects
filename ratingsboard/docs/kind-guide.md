# RatingsBoard on KIND — Full Walkthrough

## 1. Cluster + ingress + metrics-server
```bash
kind create cluster --name ratingsboard --config manifests/kind/kind-cluster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
```

## 2. Build + deploy
```bash
./manifests/kind/build-and-load.sh ratingsboard
kubectl apply -f manifests/base/
kubectl -n ratingsboard get pods -w
```

## 3. Generate ratings + view the dashboard
```bash
# rating-calc reads PanelPulse aggregates (run P3 first for real data), else seeds empty
kubectl -n ratingsboard create job --from=cronjob/rating-calc calc-now
kubectl -n ratingsboard logs job/calc-now
open http://localhost/
curl http://localhost/api/ratings/top?limit=10
```

## 4. Install monitoring + see the metrics
```bash
kubectl create namespace monitoring
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts && helm repo update
helm install kube-prom prometheus-community/kube-prometheus-stack -n monitoring
kubectl apply -f monitoring/servicemonitor.yaml
kubectl apply -f monitoring/alert.yaml
# Prometheus UI:
kubectl -n monitoring port-forward svc/kube-prom-kube-prometheus-prometheus 9090
# query: sum(rate(http_requests_total{path=~"/api/.*"}[5m]))
# Grafana:
kubectl -n monitoring port-forward svc/kube-prom-grafana 3000:80
```

## 5. See the operations controls
```bash
# ResourceQuota usage vs caps
kubectl -n ratingsboard describe resourcequota ratingsboard-quota
# LimitRange defaults (deploy a pod without requests/limits, see them filled in)
kubectl -n ratingsboard describe limitrange ratingsboard-defaults
```

## 6. Generate some load (watch RED metrics + HPA move)
```bash
kubectl -n ratingsboard run load --image=busybox -it --rm --restart=Never -- \
  sh -c "while true; do wget -q -O- http://reporting-api:8000/api/ratings/top; done"
kubectl -n ratingsboard get hpa -w
```

## Teardown
```bash
helm uninstall kube-prom -n monitoring
kind delete cluster --name ratingsboard
```
