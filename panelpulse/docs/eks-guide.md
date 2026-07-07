# PanelPulse on EKS — Deployment Guide

**Costs money — one session, then tear down (step 7).** Managed-services guide:
`manifests/eks/README-eks-datapipeline.md`.

## 1. Push images to ECR
```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text); REGION=ap-south-1
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com
for svc in collector node-agent processor aggregator; do
  aws ecr create-repository --repository-name panelpulse/$svc --region $REGION 2>/dev/null || true
  docker build -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/panelpulse/$svc:1.0 services/$svc
  docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/panelpulse/$svc:1.0
done
```

## 2. Cluster + managed infra
```bash
eksctl create cluster --name panelpulse --region $REGION \
  --nodegroup-name ng-1 --node-type t3.large --nodes 3 --managed --with-oidc
```
- **Amazon MSK** for Kafka (set `KAFKA_BOOTSTRAP` to the MSK brokers).
- **RDS Postgres** for the results DB (set `DB_HOST`; password via Secrets Manager + IRSA).

## 3. Install KEDA (lag-based processor scaling)
```bash
helm repo add kedacore https://kedacore.github.io/charts && helm repo update
helm install keda kedacore/keda -n keda --create-namespace
# then apply the ScaledObject (see manifests/base/11-processor.yaml) pointed at MSK.
```

## 4. AWS Load Balancer Controller + deploy
```bash
helm install panelpulse ./helm/panelpulse -f helm/panelpulse/values-eks.yaml \
  --set infra.inCluster=false
kubectl apply -f manifests/eks/20-ingress-alb.yaml
```
(Skip base/02-kafka.yaml and 03-results-db.yaml — use MSK + RDS.)

## 5. Verify
```bash
kubectl -n panelpulse get pods,daemonset,hpa,cronjob,ingress
# curl the ALB DNS /ingest with a sample event
```

## 6. Watch lag-based scaling
Generate ingest load; KEDA scales processors as consumer lag rises, then back down.

## 7. TEAR DOWN
```bash
kubectl delete -f manifests/eks/20-ingress-alb.yaml
helm uninstall panelpulse; helm uninstall keda -n keda
eksctl delete cluster --name panelpulse --region $REGION
# delete MSK cluster + RDS if done; verify no orphaned ALB/EBS/NAT.
```
