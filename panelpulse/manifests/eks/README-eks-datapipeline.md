# PanelPulse on EKS — managed data-pipeline services

On EKS, replace the in-cluster stateful infrastructure with managed services
(keep the cluster stateless and let AWS run the hard durable systems):

- **Kafka → Amazon MSK** (Managed Streaming for Kafka): point `KAFKA_BOOTSTRAP`
  at the MSK bootstrap brokers. MSK runs the brokers, storage, and replication —
  you don't manage the Kafka StatefulSet or its PVCs. (Alternatively Kinesis Data
  Streams, but MSK keeps the Kafka API so the code is unchanged.)
- **Results DB → Amazon RDS** (Postgres, Multi-AZ): point `DB_HOST` at the RDS
  endpoint; inject the password via IRSA + Secrets Manager.
- **Lag-based autoscaling → KEDA**: install KEDA and use a `ScaledObject` with the
  Kafka scaler so processors scale on consumer lag (the right signal). On MSK this
  reads lag from the managed brokers. See `manifests/base/11-processor.yaml` for the
  ScaledObject pattern.
- **Storage**: if you keep any in-cluster stateful component, the EBS CSI driver
  provisions the PVCs.

Drop `manifests/base/02-kafka.yaml` and `03-results-db.yaml` from the EKS apply set
(use MSK + RDS instead); keep the stateless collectors/processors/agents in-cluster.
