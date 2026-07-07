# kubernetes-portfolio

Four production-shaped Kubernetes projects, each built **local-first on KIND** then deployed to **Amazon EKS**, packaged with **Helm**. Together they exercise the full Kubernetes surface — workloads, networking, config/storage, scheduling/scaling, security, and observability — across realistic industry domains.

| # | Project | Domain | Leans into | Status |
|---|---------|--------|-----------|--------|
| **P1** | [ShopKart](shopkart/) | E-Commerce | Microservices, Ingress routing, HPA, zero-downtime deploys | ✅ Built |
| **P2** | [FinLedger](finledger/) | Finance | Security — RBAC, Pod Security, NetworkPolicy, Jobs/CronJobs | 🔜 Planned |
| **P3** | [PanelPulse](panelpulse/) | Media Measurement (ingest) | Data pipelines — Kafka StatefulSet, DaemonSet, scaling | 🔜 Planned |
| **P4** | [RatingsBoard](ratingsboard/) | Media Measurement (reporting) | Observability — Prometheus/Grafana, Helm, CronJobs, quotas | 🔜 Planned |

P3 + P4 together form an end-to-end media-measurement system (ingest → process → report).

## Per-project structure
Each project contains real service code, complete Kubernetes manifests (base + KIND + EKS overlays), a Helm chart, a KIND guide, an EKS guide (with teardown), troubleshooting, and an architecture diagram.

## Kubernetes coverage across the portfolio
- **Workloads:** Deployments, StatefulSets, DaemonSets, Jobs, CronJobs
- **Networking:** Services (ClusterIP/LoadBalancer), Ingress, NetworkPolicies, DNS discovery
- **Config/Storage:** ConfigMaps, Secrets, PV/PVC, StorageClasses
- **Scaling/Scheduling:** HPA, requests/limits, probes, affinity, PDBs
- **Security:** RBAC, ServiceAccounts (IRSA on EKS), securityContext, Pod Security Standards
- **Observability/Ops:** Prometheus, Grafana, Helm, namespaces, ResourceQuotas
- **AWS integration (EKS):** managed node groups, IRSA, AWS Load Balancer Controller, EBS/EFS CSI, ECR
