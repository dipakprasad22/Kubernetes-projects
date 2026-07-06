# RatingsBoard on EKS

- **Reports DB → Amazon RDS** (Postgres); inject the password via IRSA + Secrets Manager.
- **Source aggregates:** point `SRC_DB_*` at PanelPulse's store (its RDS, or a shared
  analytics warehouse like Redshift/Athena in a real setup).
- **Observability → Amazon Managed Prometheus + Managed Grafana** (or self-managed
  kube-prometheus-stack). The reporting-api `/metrics` and the ServiceMonitor work the same.
- **Ingress → ALB** via the AWS Load Balancer Controller.
- **Multi-tenancy:** the ResourceQuota + LimitRange apply on EKS too — combine with
  RBAC + NetworkPolicy for full tenant isolation.
Drop base/03-reports-db.yaml (use RDS); keep the stateless workloads in-cluster.
