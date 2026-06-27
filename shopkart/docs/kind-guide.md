# ShopKart on KIND — Full Walkthrough

Build and run the entire ShopKart platform locally on KIND (Kubernetes-in-Docker), free.

## Prerequisites
- Docker, `kind`, `kubectl`, (optional) `helm`
- ~4GB free RAM for the cluster + workloads

## 1. Create the cluster
```bash
kind create cluster --name shopkart --config manifests/kind/kind-cluster.yaml
kubectl get nodes        # 1 control-plane + 2 workers, Ready
```

## 2. Install the ingress controller
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller --timeout=120s
```

## 3. Install metrics-server (for HPA)
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
# KIND needs --kubelet-insecure-tls:
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl top nodes        # should return data after a minute
```

## 4. Build + load images
```bash
./manifests/kind/build-and-load.sh shopkart
```

## 5. Deploy
```bash
kubectl apply -f manifests/base/
kubectl -n shopkart get pods -w     # wait for all Running + Ready
```

## 6. Verify the platform
```bash
# data tier ready?
kubectl -n shopkart get statefulset
# services have endpoints? (the #1 networking check)
kubectl -n shopkart get endpoints
# the app, end to end:
curl http://localhost/api/catalog/products
curl -X POST http://localhost/api/users/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"a@b.com","name":"Alex","password":"secret"}'
open http://localhost/
```

## 7. Demonstrate the Kubernetes features
```bash
# Self-healing: kill a pod, watch it return
kubectl -n shopkart delete pod -l app=catalog --wait=false; kubectl -n shopkart get pods -w

# Zero-downtime rolling deploy
kubectl -n shopkart set image deploy/catalog catalog=shopkart/catalog:1.0
kubectl -n shopkart rollout status deploy/catalog

# HPA under load (run a load generator against the gateway)
kubectl -n shopkart get hpa -w
kubectl -n shopkart run load --image=busybox -it --rm --restart=Never -- \
  sh -c "while true; do wget -q -O- http://gateway:8080/api/catalog/products; done"
```

## Teardown
```bash
kind delete cluster --name shopkart
```

## Troubleshooting
See `docs/troubleshooting.md`. Quick hits:
- Pods `ImagePullBackOff` → images not loaded into KIND (`build-and-load.sh`).
- Service unreachable → `kubectl -n shopkart get endpoints <svc>` empty = selector/label or readiness.
- HPA `<unknown>` → metrics-server missing or needs `--kubelet-insecure-tls`.
- NetworkPolicies don't block → default KIND CNI doesn't enforce them (install Calico to see enforcement).
