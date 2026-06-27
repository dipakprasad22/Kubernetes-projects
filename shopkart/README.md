# ShopKart — E-Commerce Microservices Platform on Kubernetes

A production-shaped e-commerce platform built as microservices on Kubernetes. Built **local-first on KIND**, then deployed to **Amazon EKS**, packaged with **Helm**. Demonstrates the microservices, networking, scaling, and operational patterns of a real online retail system.

> Part of the [kubernetes-portfolio](../) — Project 1 of 4.

---

## What this project demonstrates

| Kubernetes concept | Where it shows up in ShopKart |
|---|---|
| **Deployments + rolling updates** | All 6 services; `maxUnavailable: 0, maxSurge: 1` for zero-downtime deploys |
| **Services + DNS discovery** | Gateway calls `http://catalog`, `http://orders` etc. by Service name |
| **Ingress (path routing)** | One entry point: `/` → web, `/api/*` → gateway (→ ALB on EKS) |
| **ConfigMaps + Secrets** | Shared config + DB credentials (IRSA/Secrets Manager on EKS) |
| **HorizontalPodAutoscaler** | Every service scales 2→8/10 on CPU (flash-sale traffic) |
| **StatefulSets** | Postgres + Redis data tier (local); managed services on EKS |
| **Readiness / liveness probes** | `/ready` verifies DB/Redis; `/health` for liveness — honest rollouts |
| **NetworkPolicy** | Default-deny + allowlist; data tier reachable only by its consumers |
| **securityContext** | Non-root, drop ALL caps, read-only root FS, no privilege escalation |
| **PodDisruptionBudget** | Gateway protected during node drains |
| **Helm** | The whole platform packaged as a chart with per-environment values |

---

## Architecture

Users hit one **Ingress** → routed by path to the **web** frontend (UI) and the **API gateway**. The gateway proxies `/api/<service>/*` to the backend microservices (**catalog, cart, orders, users**) by their Kubernetes Service DNS names. Backends use a data tier — **Postgres** (catalog/orders/users) and **Redis** (cart sessions); **orders** publishes `order.created` events to **RabbitMQ** for async downstream processing.

```
Users → Ingress (ALB on EKS)
          ├── /        → web (Node.js BFF)
          └── /api/*   → gateway (Go)
                          ├── http://catalog → Postgres
                          ├── http://cart    → Redis
                          ├── http://orders  → Postgres + RabbitMQ
                          └── http://users   → Postgres
```

See `docs/architecture.png` (export the system-design diagram) for the full picture.

---

## Services

| Service | Language | Role | Backing store |
|---|---|---|---|
| `web` | Node.js | Storefront UI + BFF (proxies to gateway) | — |
| `gateway` | Go | API gateway — routes `/api/<svc>` to backends | — |
| `catalog` | Go | Product catalog | Postgres |
| `cart` | Go | Shopping carts (24h TTL) | Redis |
| `orders` | Go | Order placement + events | Postgres + RabbitMQ |
| `users` | Go | Accounts + auth (bcrypt) | Postgres |

Go services build to **distroless, non-root, static** images (tiny, minimal attack surface).

---

## Quick start (KIND)

```bash
# 1. Create the cluster
kind create cluster --name shopkart --config manifests/kind/kind-cluster.yaml

# 2. Install the nginx ingress controller
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait --namespace ingress-nginx --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller --timeout=120s

# 3. Build images and load them into KIND
./manifests/kind/build-and-load.sh shopkart

# 4. Deploy (raw manifests)
kubectl apply -f manifests/base/

# 5. Verify
kubectl -n shopkart get pods,svc,ingress,hpa
curl http://localhost/api/catalog/products      # products via the gateway
open http://localhost/                            # the storefront UI
```

Or with **Helm**:
```bash
helm install shopkart ./helm/shopkart -f helm/shopkart/values-kind.yaml
```

Full walkthrough: [`docs/kind-guide.md`](docs/kind-guide.md).

---

## Deploy to EKS

See [`docs/eks-guide.md`](docs/eks-guide.md). Summary: push images to ECR, create the cluster with `eksctl --with-oidc`, install the AWS Load Balancer Controller, swap the Ingress for the ALB overlay, point the data tier at RDS/ElastiCache (via IRSA + Secrets Manager), then `helm install -f values-eks.yaml`. **Tear down the same day** (delete Ingress/PVCs before the cluster).

---

## Repository layout

```
p1-shopkart/
├── services/          # real Go microservices (gateway, catalog, cart, orders, users)
├── web/               # Node.js storefront/BFF
├── manifests/
│   ├── base/          # core K8s manifests (Deployments, Services, HPA, Ingress, NetPol, PDB)
│   ├── kind/          # KIND cluster config + image build/load script
│   └── eks/           # EKS overlay (ALB Ingress, IRSA SA, managed data tier notes)
├── helm/shopkart/     # Helm chart (templated, per-environment values)
└── docs/              # architecture diagram, KIND guide, EKS guide, troubleshooting
```

---

## Production notes

- **Data tier:** in-cluster StatefulSets are for local/KIND only. On EKS, use **RDS** (Postgres) and optionally **ElastiCache** (Redis) — keep the cluster stateless; inject DB credentials via **IRSA + Secrets Manager**, not a Kubernetes Secret.
- **Secrets:** the committed `DB_PASSWORD` is a dev placeholder. Never commit real secrets; on EKS enable etcd encryption-at-rest (KMS) and source from Secrets Manager.
- **Scaling:** HPAs need `metrics-server`. The gateway scales to 10 replicas; backends to 8 — tune from real load.
- **Zero-downtime:** every Deployment uses `maxUnavailable: 0` + readiness probes, so deploys never drop traffic.
