# FinLedger on EKS — Security Hardening Checklist

Finance workloads require defense in depth. On EKS, layer AWS controls onto the
Kubernetes controls already in the base manifests:

- [ ] **etcd encryption-at-rest (KMS):** enable at cluster creation so Secrets are
      encrypted in etcd, not just base64. `eksctl ... --with-oidc` + a KMS key, or
      enable `secretsEncryption` in the cluster config.
- [ ] **IRSA for DB credentials:** the transaction-api reads its DB password from
      Secrets Manager via a least-privilege IAM role (`manifests/eks/10-irsa-...`),
      not a Kubernetes Secret.
- [ ] **RDS Multi-AZ** for the ledger store (not an in-cluster StatefulSet) — managed
      backups, failover, point-in-time recovery for the source of truth.
- [ ] **Private API endpoint + private node subnets** — no public exposure of the
      control plane or nodes.
- [ ] **Pod Security "restricted"** (already enforced via the namespace label) — verify
      it holds on EKS.
- [ ] **NetworkPolicies enforced** — install Calico/Cilium (the default VPC CNI needs
      the Calico add-on for NetworkPolicy enforcement), and additionally use security
      groups for pods for an extra AWS-layer firewall.
- [ ] **ALB TLS** via an ACM certificate; consider mTLS for client auth.
- [ ] **Audit:** enable EKS control-plane audit logging to CloudWatch.

Defense in depth: RBAC (access) + Pod Security (workload) + NetworkPolicy (network)
+ KMS/IRSA (data) + RDS (durability) + audit logging — no single control alone.
