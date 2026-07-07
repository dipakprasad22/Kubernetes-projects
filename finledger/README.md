# FinLedger — Finance Transaction Platform on Kubernetes

A security-hardened financial transaction platform on Kubernetes. Built **local-first on KIND**, then deployed to **Amazon EKS**, packaged with **Helm**. Where ShopKart leaned into microservices, FinLedger leans into **security, correctness, and compliance** — the concerns that define how finance actually runs on Kubernetes.

> Part of the [kubernetes-portfolio](../) — Project 2 of 4. **The security-architecture artifact.**

---

## What this project demonstrates

| Kubernetes concept | Where it shows up in FinLedger |
|---|---|
| **RBAC + ServiceAccounts** | Least-privilege auditor SA (read-only, **no** secret access); per-workload SAs; IRSA on EKS |
| **Pod Security Standards** | Namespace enforces `restricted` — API server rejects root/privileged pods |
| **securityContext** | Every pod: non-root, drop ALL caps, read-only root FS, no priv-esc, seccomp RuntimeDefault |
| **NetworkPolicy (zero-trust)** | **Default-deny** ingress+egress, then allowlist only required flows (incl. DNS egress) |
| **Secrets hardening** | base64 caveat documented; KMS etcd encryption + Secrets Manager (IRSA) on EKS |
| **@Transactional atomicity** | Money moves are all-or-nothing — double-entry ledger, debit+credit commit together |
| **Jobs / CronJobs** | Reconciliation CronJob verifies ledger integrity (debits == credits) on a schedule |
| **Worker pattern** | Fraud-check worker has **no web port** — liveness via process check, async event consumer |
| **StatefulSet** | Postgres ledger store (local); RDS Multi-AZ on EKS |
| **PodDisruptionBudget** | `minAvailable: 2` keeps the transaction API available during node drains |
| **Probes** | Startup probe gates liveness for the slow JVM (the K5 fix); readiness for honest rollouts |

---

## Correctness: the double-entry ledger

Every transfer creates **two** immutable ledger entries — a debit (negative) on the source account and a credit (positive) on the destination — inside a single `@Transactional` boundary, so a partial money movement can never occur. Across the whole ledger, the sum of all signed entries must be exactly **zero**. The **reconciliation CronJob** verifies this invariant on a schedule and exits non-zero (failing the Job) if the ledger is ever out of balance — turning a correctness guarantee into an alertable signal.

```
POST /api/transactions {from, to, amount}
   └─ @Transactional:
        save Transfer
        LedgerEntry(from, -amount)   ← debit
        LedgerEntry(to,   +amount)   ← credit
        (any failure → full rollback)
   └─ emit transaction.created → fraud-worker (async)
```

---

## Security: defense in depth

FinLedger applies the K7 security model as layered controls — no single one isolates the workload:

1. **RBAC** — least-privilege access; the auditor SA can read workloads but **not** secrets.
2. **Pod Security `restricted`** — enforced at the namespace; non-compliant pods are rejected.
3. **securityContext** — non-root, read-only FS, dropped capabilities on every container.
4. **NetworkPolicy** — default-deny zero-trust, then explicit allowlist (API←ingress, apps→DB, DB←apps, DNS egress).
5. **Secrets** — KMS etcd encryption + Secrets Manager via IRSA on EKS (not stored keys).
6. **PDB** — availability protected during maintenance.

See `manifests/eks/README-eks-security.md` for the full EKS hardening checklist.

---

## Services

| Service | Language | Role | Pattern |
|---|---|---|---|
| `transaction-api` | Java / Spring Boot | Accept transfers, write atomic double-entry postings | Web service, 3 replicas, PDB, HPA |
| `fraud-worker` | Java / Spring Boot | Score transactions for fraud (async) | **Worker — no web port** |
| `reconciliation` | Java / Spring Boot | Verify ledger integrity on schedule | **CronJob — run-to-completion** |
| `postgres` | (image) | Ledger source of truth | StatefulSet (local) / RDS (EKS) |

---

## Quick start (KIND)

```bash
kind create cluster --name finledger --config manifests/kind/kind-cluster.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=120s
./manifests/kind/build-and-load.sh finledger
kubectl apply -f manifests/base/
kubectl -n finledger get pods,svc,cronjob,netpol

# Try a transfer (account 0 is the funding account, allowed to go negative):
curl -X POST http://localhost/api/transactions -H 'Content-Type: application/json' \
  -d '{"fromAccount":0,"toAccount":1,"amount":100.00}'
curl http://localhost/api/accounts/1/balance      # {"account":1,"balance":100.0000}

# Run reconciliation on demand:
kubectl -n finledger create job --from=cronjob/reconciliation recon-now
kubectl -n finledger logs job/recon-now           # "reconciliation OK: ledger balanced"
```

Full walkthrough: [`docs/kind-guide.md`](docs/kind-guide.md). EKS: [`docs/eks-guide.md`](docs/eks-guide.md).

---

## Demonstrating the security controls

```bash
# Pod Security: a non-compliant (root) pod is REJECTED in this namespace
kubectl -n finledger run bad --image=nginx          # error: violates "restricted"

# RBAC: the auditor can read pods but NOT secrets
kubectl auth can-i list pods    -n finledger --as=system:serviceaccount:finledger:finledger-auditor   # yes
kubectl auth can-i get secrets  -n finledger --as=system:serviceaccount:finledger:finledger-auditor   # no

# NetworkPolicy (needs Calico to enforce): a random pod cannot reach Postgres
```

---

## Production notes

- **Data tier:** in-cluster Postgres is local-only. On EKS use **RDS Multi-AZ** (managed backups/failover for the ledger source of truth) and inject the password via **IRSA + Secrets Manager**.
- **NetworkPolicy enforcement:** the default KIND/EKS-VPC-CNI needs **Calico/Cilium** to actually enforce policies — without it they're inert. Install Calico to see zero-trust in action.
- **Secrets:** the committed `DB_PASSWORD` is a dev placeholder — never commit real secrets.
- **mTLS / audit:** for production finance, add mTLS at the ALB and enable EKS control-plane audit logging.
