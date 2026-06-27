# EKS Data Tier — use managed services, keep the cluster stateless

On EKS, do NOT run Postgres as an in-cluster StatefulSet for production. Instead:

- **Postgres → Amazon RDS** (Multi-AZ): point services at the RDS endpoint via the
  ConfigMap `DB_HOST`, and inject the password from **Secrets Manager** using **IRSA**
  (the `shopkart-sa` ServiceAccount) rather than a Kubernetes Secret.
- **Redis → Amazon ElastiCache** (optional): point `REDIS_HOST` at the ElastiCache endpoint.
- Remove `manifests/base/02-postgres.yaml` and `03-redis.yaml` from the EKS apply set;
  keep only the stateless workloads in the cluster.

This is the "same app, three ways" judgment: the data tier stays in managed services
across EC2/ECS/EKS — only the compute orchestration changes.

If you *do* need in-cluster persistent storage on EKS, the EBS CSI driver provisions
EBS volumes for `ReadWriteOnce` PVCs (install the `aws-ebs-csi-driver` EKS add-on);
use EFS (RWX) for shared read-write.
