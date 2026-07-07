# FinLedger on KIND — Full Walkthrough

## Prerequisites
Docker, kind, kubectl, (optional) helm. ~4GB RAM. The Maven build runs inside the
Docker multi-stage build, so you don't need Maven installed locally.

## 1. Cluster + ingress + metrics-server
```bash
kind create cluster --name finledger --config manifests/kind/kind-cluster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p='[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
```

## 2. (Recommended) Install Calico so NetworkPolicies are enforced
The default KIND CNI ignores NetworkPolicies. To SEE the zero-trust policies work,
install Calico (or accept that policies apply but aren't enforced on the default CNI).

## 3. Build + deploy
```bash
./manifests/kind/build-and-load.sh finledger
kubectl apply -f manifests/base/
kubectl -n finledger get pods -w     # wait for Running + Ready (JVM startup ~30-40s)
```

## 4. Exercise the platform
```bash
# fund account 1 from the funding account (id 0)
curl -X POST http://localhost/api/transactions -H 'Content-Type: application/json' \
  -d '{"fromAccount":0,"toAccount":1,"amount":500.00}'
# move money 1 -> 2
curl -X POST http://localhost/api/transactions -H 'Content-Type: application/json' \
  -d '{"fromAccount":1,"toAccount":2,"amount":200.00}'
# balances
curl http://localhost/api/accounts/1/balance     # 300.0000
curl http://localhost/api/accounts/2/balance     # 200.0000
# insufficient funds is rejected atomically (409)
curl -i -X POST http://localhost/api/transactions -H 'Content-Type: application/json' \
  -d '{"fromAccount":2,"toAccount":1,"amount":999999.00}'
```

## 5. Reconciliation (the integrity check)
```bash
kubectl -n finledger create job --from=cronjob/reconciliation recon-now
kubectl -n finledger logs job/recon-now
# "reconciliation OK: ledger balanced (sum == 0)"  — because every debit has a credit
```

## 6. Prove the security controls
```bash
# Pod Security "restricted" rejects a root pod:
kubectl -n finledger run bad --image=nginx     # FORBIDDEN

# RBAC least-privilege (auditor can read pods, NOT secrets):
kubectl auth can-i list pods   -n finledger --as=system:serviceaccount:finledger:finledger-auditor
kubectl auth can-i get secrets -n finledger --as=system:serviceaccount:finledger:finledger-auditor

# securityContext: the container runs non-root with a read-only FS:
kubectl -n finledger exec deploy/transaction-api -- id           # uid=1001, non-root
kubectl -n finledger exec deploy/transaction-api -- touch /x 2>&1 || echo "read-only FS"
```

## Teardown
```bash
kind delete cluster --name finledger
```
