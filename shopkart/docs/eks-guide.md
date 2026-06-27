# ShopKart on EKS — Deployment Guide

Deploy ShopKart to Amazon EKS. **Costs real money — do it in one session and tear down (step 9).**

## Prerequisites
- `eksctl`, `kubectl`, `aws` CLI, `helm`, Docker
- ECR repos for each image; an RDS Postgres (and optionally ElastiCache Redis)

## 1. Push images to ECR
```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text); REGION=ap-south-1
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com
for svc in gateway catalog cart orders users web; do
  aws ecr create-repository --repository-name shopkart/$svc --region $REGION 2>/dev/null || true
  docker build -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/shopkart/$svc:1.0 \
    $( [ "$svc" = web ] && echo web || echo services/$svc )
  docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/shopkart/$svc:1.0
done
```

## 2. Create the cluster (with OIDC for IRSA)
```bash
eksctl create cluster --name shopkart --region $REGION \
  --nodegroup-name ng-1 --node-type t3.medium --nodes 2 --nodes-min 2 --nodes-max 4 \
  --managed --with-oidc
```

## 3. Install the AWS Load Balancer Controller
```bash
# create its IRSA role, then:
helm repo add eks https://aws.github.io/eks-charts && helm repo update
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system --set clusterName=shopkart \
  --set serviceAccount.create=false --set serviceAccount.name=aws-load-balancer-controller
```

## 4. Set up the data tier (RDS) + IRSA
- Create an RDS Postgres (Multi-AZ); create the `catalog`/`orders`/`users` databases.
- Store the DB password in Secrets Manager.
- Create the IRSA ServiceAccount (`manifests/eks/50-irsa-serviceaccount.yaml` + eksctl).

## 5. Install metrics-server / EBS CSI add-on (if needed)
```bash
eksctl create addon --name metrics-server --cluster shopkart 2>/dev/null || true
eksctl create addon --name aws-ebs-csi-driver --cluster shopkart 2>/dev/null || true
```

## 6. Deploy via Helm (EKS values)
```bash
# edit helm/shopkart/values-eks.yaml: ECR registry, RDS endpoint, ElastiCache endpoint
helm install shopkart ./helm/shopkart -f helm/shopkart/values-eks.yaml \
  --set datatier.inCluster=false
kubectl apply -f manifests/eks/20-ingress-alb.yaml
```

## 7. Verify
```bash
kubectl -n shopkart get pods,svc,ingress
kubectl -n shopkart get ingress shopkart -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'
# curl the ALB DNS name /api/catalog/products
```

## 8. IRSA + ALB highlights
- Pods that read the DB secret use the `shopkart-sa` ServiceAccount (IRSA → IAM role scoped to the one secret) — no stored AWS keys.
- The Ingress becomes a real ALB with path routing (target-type IP, VPC CNI pod IPs).
- PVCs (if used) provision EBS volumes via the EBS CSI driver.

## 9. TEAR DOWN (don't skip — avoid surprise bills)
```bash
kubectl delete -f manifests/eks/20-ingress-alb.yaml    # deletes the ALB
helm uninstall shopkart
eksctl delete cluster --name shopkart --region $REGION
# verify in console: no ALB, no EBS volumes, no NAT gateway lingering; delete RDS if done.
```
