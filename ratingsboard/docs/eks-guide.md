# RatingsBoard on EKS — Deployment Guide

**Costs money — one session, tear down (step 6).** See manifests/eks/README-eks.md.

## 1. Push images to ECR
```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text); REGION=ap-south-1
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com
docker build -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/ratingsboard/reporting-api:1.0 services/reporting-api
docker build -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/ratingsboard/rating-calc:1.0 services/rating-calc
docker build -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/ratingsboard/dashboard:1.0 dashboard
for i in reporting-api rating-calc dashboard; do
  aws ecr create-repository --repository-name ratingsboard/$i --region $REGION 2>/dev/null || true
  docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/ratingsboard/$i:1.0
done
```

## 2. Cluster + RDS + IRSA
```bash
eksctl create cluster --name ratingsboard --region $REGION \
  --nodegroup-name ng-1 --node-type t3.medium --nodes 2 --managed --with-oidc
```
- RDS Postgres for the reports DB; password in Secrets Manager; IRSA SA (manifests/eks/10-...).
- Point SRC_DB_* at PanelPulse's store / shared warehouse.

## 3. Observability
Use **Amazon Managed Prometheus + Managed Grafana**, or install kube-prometheus-stack.
Apply the ServiceMonitor + alert (monitoring/).

## 4. ALB controller + deploy
```bash
helm install ratingsboard ./helm/ratingsboard -f helm/ratingsboard/values-eks.yaml --set infra.inCluster=false
kubectl apply -f manifests/eks/20-ingress-alb.yaml
```

## 5. Verify
```bash
kubectl -n ratingsboard get pods,hpa,cronjob,ingress,resourcequota
# curl the ALB DNS / and /api/ratings/top
```

## 6. TEAR DOWN
```bash
kubectl delete -f manifests/eks/20-ingress-alb.yaml
helm uninstall ratingsboard
eksctl delete cluster --name ratingsboard --region $REGION
# delete RDS if done; verify no orphaned ALB/EBS/NAT.
```
