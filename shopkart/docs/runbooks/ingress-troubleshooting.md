# Runbook: Troubleshooting NGINX Ingress Connectivity in KIND Cluster

## Purpose

This runbook documents the troubleshooting process for resolving the issue where requests to the application through the NGINX Ingress returned:

```text
curl: (56) Recv failure: Connection reset by peer
```

The goal is to provide a structured approach for diagnosing similar Kubernetes networking and Ingress issues.

---

# Environment

* Kubernetes: KIND (Multi-node)
* Ingress Controller: NGINX Ingress Controller v1.15.1
* Runtime: Docker Desktop (macOS)
* Application: ShopKart Microservices

---

# Problem Statement

The backend services were healthy and accessible internally, but requests through the Ingress failed.

Example:

```bash
curl http://localhost/api/catalog/products
```

Result:

```text
curl: (56) Recv failure: Connection reset by peer
```

---

# Troubleshooting Steps

## Step 1: Verify Pod Status

Check whether all application pods are running.

```bash
kubectl -n shopkart get pods
```

Expected:

* All application pods should be in Running state.
* No CrashLoopBackOff or ImagePullBackOff.

---

## Step 2: Verify Services

Check whether Kubernetes Services exist.

```bash
kubectl -n shopkart get svc
```

Expected:

* gateway
* catalog
* users
* cart
* orders
* web
* postgres
* redis

---

## Step 3: Verify Endpoints

Ensure Services have healthy backend Pods.

```bash
kubectl -n shopkart get endpoints
```

Expected:

Each Service should have Pod IPs assigned.

Example:

```text
gateway
10.244.1.5:8080
10.244.2.6:8080
```

If no endpoints exist:

* Pod labels
* Service selectors
* Readiness probes

should be verified.

---

## Step 4: Verify Gateway Directly

Port-forward the Gateway Service.

```bash
kubectl -n shopkart port-forward svc/gateway 8080:8080
```

Test:

```bash
curl http://localhost:8080/health
```

```bash
curl http://localhost:8080/api/catalog/products
```

Expected:

```http
HTTP/1.1 200 OK
```

Purpose:

Confirms that the application itself is healthy before troubleshooting Ingress.

---

## Step 5: Verify Web Service

Inspect the Web Service.

```bash
kubectl -n shopkart get svc web -o yaml
```

Purpose:

Verify:

* Service Port
* Target Port
* Selector

---

## Step 6: Verify Ingress

```bash
kubectl -n shopkart get ingress
```

```bash
kubectl -n shopkart describe ingress shopkart
```

Purpose:

Verify:

* Ingress Class
* Backend Services
* Service Ports
* Paths
* Rewrite Rules

---

## Step 7: Verify Ingress Controller

```bash
kubectl get pods -n ingress-nginx
```

Purpose:

Ensure the controller is running.

---

## Step 8: Review Ingress Controller Logs

```bash
kubectl -n ingress-nginx logs deploy/ingress-nginx-controller --tail=100
```

Purpose:

Look for:

* Backend reloads
* Endpoint errors
* Routing failures
* RBAC issues

---

## Step 9: Inspect Generated NGINX Configuration

```bash
kubectl -n ingress-nginx exec deploy/ingress-nginx-controller -- nginx -T
```

Purpose:

Verify:

* Generated location blocks
* Rewrite rules
* Upstream services

---

## Step 10: Test from Inside the Cluster

Launch a temporary Curl Pod.

```bash
kubectl run curl --rm -it --restart=Never --image=curlimages/curl -- sh
```

Test Gateway:

```bash
curl http://gateway.shopkart.svc.cluster.local:8080/health
```

```bash
curl http://gateway.shopkart.svc.cluster.local:8080/api/catalog/products
```

Test Ingress:

```bash
curl http://ingress-nginx-controller.ingress-nginx.svc.cluster.local/api/catalog/products
```

Purpose:

This isolates Kubernetes networking from Docker Desktop networking.

---

## Step 11: Verify Network Policies

List policies.

```bash
kubectl -n shopkart get networkpolicy
```

Inspect policies.

```bash
kubectl -n shopkart describe networkpolicy
```

Temporary test:

```bash
kubectl -n shopkart delete networkpolicy --all
```

Purpose:

Rule out traffic being blocked by NetworkPolicy.

---

## Step 12: Verify Docker Port Mapping

```bash
docker port shopkart-control-plane
```

Expected:

```text
80/tcp -> 0.0.0.0:80
443/tcp -> 0.0.0.0:443
```

Purpose:

Confirm host ports are published.

---

## Step 13: Test from Control Plane

```bash
docker exec -it shopkart-control-plane curl http://localhost
```

Result:

```text
Connection refused
```

Purpose:

Showed nothing was listening on port 80 inside the control-plane container.

---

## Step 14: Check Listening Ports

```bash
docker exec -it shopkart-control-plane ss -ltnp
```

Purpose:

Verify whether anything is listening on:

```text
0.0.0.0:80
```

Result:

No listener found.

---

## Step 15: Verify Ingress Controller Scheduling

```bash
kubectl -n ingress-nginx get pods -o wide
```

Result:

```text
NODE
shopkart-worker2
```

The controller was running on the worker node instead of the control-plane node.

---

## Step 16: Verify Deployment Configuration

```bash
kubectl -n ingress-nginx get deploy ingress-nginx-controller -o yaml
```

Purpose:

Check:

* hostPort configuration
* nodeSelector
* tolerations

Observation:

No nodeSelector existed to force scheduling onto the control-plane node.

---

## Step 17: Verify Cluster Nodes

```bash
kubectl get nodes -o wide
```

Purpose:

Identify the control-plane hostname.

Example:

```text
shopkart-control-plane
```

---

## Step 18: Fix the Scheduling

Patch the Deployment.

```bash
kubectl patch deployment ingress-nginx-controller \
-n ingress-nginx \
--type merge \
-p '{
  "spec":{
    "template":{
      "spec":{
        "nodeSelector":{
          "kubernetes.io/hostname":"shopkart-control-plane"
        }
      }
    }
  }
}'
```

Monitor rollout.

```bash
kubectl -n ingress-nginx get pods -o wide -w
```

Expected:

```text
NODE
shopkart-control-plane
```

---

## Step 19: Validate the Fix

```bash
curl http://localhost/api/catalog/products
```

Expected:

```json
[
  {
    "id":1,
    "name":"Wireless Mouse"
  }
]
```

Issue resolved.

---

# Root Cause Analysis (RCA)

## Problem

The NGINX Ingress Controller Pod was scheduled on a worker node while Docker Desktop exposed ports 80 and 443 only on the KIND control-plane node.

Traffic Flow:

```text
Client
   │
   ▼
localhost:80
   │
Docker Port Mapping
   │
Control Plane Node
   │
(No Ingress Controller)
   │
Connection Reset
```

Although the Ingress Controller was healthy, it was listening on a different Kubernetes node.

As a result:

* Requests from inside Kubernetes succeeded.
* Requests through localhost failed.
* Docker accepted the TCP connection but no process was listening on port 80 inside the control-plane container.

---

# Resolution

Force the Ingress Controller Deployment to run on the control-plane node.

```yaml
spec:
  template:
    spec:
      nodeSelector:
        kubernetes.io/hostname: shopkart-control-plane
```

After rescheduling:

* NGINX listened on host port 80.
* Docker port mapping became functional.
* All application routes were accessible via localhost.

---

# Lessons Learned

* Always verify Pods, Services, Endpoints, and Ingress before investigating networking.
* Test connectivity from inside the cluster to isolate Kubernetes issues from host networking.
* Verify Docker host port mappings when using KIND.
* In multi-node KIND clusters, ensure the Ingress Controller runs on the control-plane node when using `hostPort` or host port mappings.
* Prefer simple `Prefix` path rules unless regex-based routing is explicitly required.
* Keep troubleshooting layered: Application → Service → Endpoints → Ingress → NetworkPolicy → Docker Networking → Node Scheduling.

---

# Useful Commands Reference

```bash
kubectl get pods -A

kubectl get svc -A

kubectl get endpoints -A

kubectl get ingress -A

kubectl describe ingress <name>

kubectl logs -n ingress-nginx deploy/ingress-nginx-controller

kubectl exec -n ingress-nginx deploy/ingress-nginx-controller -- nginx -T

kubectl run curl --rm -it --restart=Never --image=curlimages/curl -- sh

docker port shopkart-control-plane

docker exec -it shopkart-control-plane ss -ltnp

kubectl get nodes -o wide

kubectl get pods -n ingress-nginx -o wide
```
