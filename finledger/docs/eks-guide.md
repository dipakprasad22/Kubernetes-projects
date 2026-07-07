# FinLedger on EKS — Deployment Guide

**Costs money — one session, then tear down (step 8).** Finance hardening checklist:
`manifests/eks/README-eks-security.md`.

## 1. Push images to ECR
```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text); REGION=ap-south-1
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT.dkr.ecr.$REGION.amazonaws.com
for svc in transaction-api fraud-worker reconciliation; do
  aws ecr create-repository --repository-name finledger/$svc --region $REGION 2>/dev/null || true
  docker build -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/finledger/$svc:1.0 services/$svc
  docker push $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/finledger/$svc:1.0
done
```

## 2. Create the cluster WITH secrets encryption (KMS) + OIDC
```bash
# Create a KMS key, then a cluster config with secretsEncryption + OIDC enabled.
eksctl create cluster --name finledger --region $REGION \
  --nodegroup-name ng-1 --node-type t3.medium --nodes 2 --managed --with-oidc
# (For real finance: use a cluster config file with secretsEncryption.keyARN set.)
```

## 3. RDS + IRSA + Secrets Manager
- Create RDS Postgres (Multi-AZ); create the `finledger` database.
- Store the DB password in Secrets Manager.
- Create the IRSA SA (manifests/eks/10-irsa-serviceaccount.yaml + eksctl iamserviceaccount),
  scoped to read ONLY that secret.

## 4. Install Calico (NetworkPolicy enforcement) + ALB controller
```bash
# Calico for NetworkPolicy; AWS Load Balancer Controller for the ALB Ingress.
```

## 5. Deploy
```bash
helm install finledger ./helm/finledger -f helm/finledger/values-eks.yaml
kubectl apply -f manifests/eks/40-ingress-alb.yaml
```

## 6. Verify security posture
```bash
kubectl -n finledger get pods,netpol
kubectl auth can-i get secrets -n finledger --as=system:serviceaccount:finledger:finledger-auditor  # no
# confirm Secrets are KMS-encrypted in etcd (cluster secretsEncryption enabled)
```

## 7. Audit logging
Enable EKS control-plane audit logs to CloudWatch (cluster logging config).

## 8. TEAR DOWN
```bash
kubectl delete -f manifests/eks/40-ingress-alb.yaml    # ALB
helm uninstall finledger
eksctl delete cluster --name finledger --region $REGION
# delete RDS + the KMS key + Secrets Manager secret if done; verify no orphaned ALB/EBS/NAT.
```
