# ShopKart on Amazon EKS - Deployment & Troubleshooting Runbook

## Project Overview

This project deploys the ShopKart microservices application on Amazon EKS using Kubernetes and Helm.

### Architecture

```
Developer
    │
Docker Build
    │
Amazon ECR
    │
Amazon EKS
    │
 ┌─────────────────────────────┐
 │ Gateway │ Catalog │ Orders │
 │ Users   │ Cart    │ Web    │
 └─────────────────────────────┘
        │
        ├── Amazon RDS PostgreSQL
        ├── Amazon ElastiCache Redis
        └── AWS ALB Ingress Controller
```

---

# Technologies Used

- Amazon EKS
- Docker
- Amazon ECR
- Helm
- Kubernetes
- Amazon RDS PostgreSQL
- Amazon ElastiCache Redis Serverless
- AWS Load Balancer Controller
- IAM Roles for Service Accounts (IRSA)
- AWS IAM
- Go Microservices

---

# Deployment Steps

## 1. Build and Push Docker Images

```bash
ACCOUNT=$(aws sts get-caller-identity --query Account --output text)
REGION=us-east-1

aws ecr get-login-password --region $REGION \
| docker login \
--username AWS \
--password-stdin \
$ACCOUNT.dkr.ecr.$REGION.amazonaws.com

for svc in gateway catalog cart orders users web
do
    aws ecr create-repository \
      --repository-name shopkart/$svc \
      --region $REGION || true

    docker build \
      --platform linux/amd64 \
      -t $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/shopkart/$svc:1.0 \
      $( [ "$svc" = web ] && echo web || echo services/$svc )

    docker push \
      $ACCOUNT.dkr.ecr.$REGION.amazonaws.com/shopkart/$svc:1.0
done
```

---

## 2. Create Amazon EKS Cluster

```bash
eksctl create cluster \
--name shopkart \
--region us-east-1 \
--managed \
--nodegroup-name ng-1 \
--node-type t2.medium \
--nodes 2 \
--with-oidc
```

---

## 3. Install AWS Load Balancer Controller

Create IAM Policy

Create IAM Role (IRSA)

Install Helm chart

Verify:

```bash
kubectl get deployment -n kube-system
kubectl get pods -n kube-system
```

---

## 4. Create Amazon RDS PostgreSQL

Create databases

```
catalog
orders
users
```

Verify

```sql
SELECT datname FROM pg_database;
```

---

## 5. Create ElastiCache Redis Serverless

Enable

- Encryption in transit
- Same VPC as EKS

---

## 6. Update Helm values

Update

- ECR Image Registry
- RDS Endpoint
- Redis Endpoint
- Image Tag

---

## 7. Deploy

```bash
helm install shopkart ./helm/shopkart \
-f helm/shopkart/values-eks.yaml
```

---

# Troubleshooting Log

---

# Issue 1

## Docker Build Failed

### Error

```
unable to prepare context
services/gateway not found
```

### Diagnosis

Verified project directory structure.

```
ls
find .
```

Root cause:

Wrong Docker build context.

### Fix

Corrected build path.

```
services/<service-name>
```

---

# Issue 2

## ECR Repository Does Not Exist

### Error

```
repository does not exist
```

### Diagnosis

Listed repositories

```bash
aws ecr describe-repositories
```

### Root Cause

Repositories were deleted.

### Fix

Created repositories before pushing images.

---

# Issue 3

## Helm Installation Failed

### Error

```
namespace shopkart not found
```

### Diagnosis

```bash
kubectl get ns
```

### RCA

Namespace was never created.

### Fix

```bash
kubectl create namespace shopkart
```

---

# Issue 4

## Helm Release Already Exists

### Error

```
cannot reuse a name
```

### Diagnosis

```bash
helm list -A
```

### Fix

```
helm uninstall
```

or

```
helm upgrade
```

---

# Issue 5

## AWS Load Balancer Controller Pods Not Created

### Error

```
no endpoints available
```

### Diagnosis

```
kubectl describe deployment
kubectl describe rs
kubectl get sa
```

### RCA

ServiceAccount missing.

Controller installed using

```
serviceAccount.create=false
```

without creating ServiceAccount.

### Fix

Created IAM Role + ServiceAccount using IRSA.

---

# Issue 6

## ImagePullBackOff

### Error

```
no match for platform in manifest
```

### Diagnosis

```
kubectl describe pod
```

### RCA

Docker images built for ARM64 (Mac M-series).

EKS worker nodes were AMD64.

### Fix

```
docker build \
--platform linux/amd64
```

---

# Issue 7

## InvalidImageName

### Error

```
<ACCOUNT_ID>.dkr.ecr...
```

### RCA

Placeholder never replaced.

### Fix

Updated values-eks.yaml

```
imageRegistry:
```

with actual AWS Account ID.

---

# Issue 8

## Orders and Users CrashLoopBackOff

### Error

```
lookup <rds-endpoint>
```

### Diagnosis

```
kubectl logs
kubectl get configmap
```

### RCA

ConfigMap still had placeholder values.

### Fix

Updated

```
DB_HOST
```

to actual RDS endpoint.

---

# Issue 9

## PostgreSQL Authentication Failed

### Error

```
no pg_hba.conf entry
```

### RCA

SSL disabled.

Amazon RDS requires SSL.

### Fix

```
DB_SSLMODE=require
```

---

# Issue 10

## ALB Not Created

### Error

```
AccessDenied
DescribeLoadBalancers
```

### Diagnosis

```
kubectl describe ingress
```

### RCA

IAM Role missing permissions.

### Fix

Attached AWSLoadBalancerControllerIAMPolicy.

---

# Issue 11

## Web Service Missing

### Error

```
service web not found
```

### Diagnosis

```
kubectl get svc
```

### RCA

Helm chart deployed only backend services.

### Fix

Added Deployment + Service for Web frontend.

---

# Issue 12

## Cart Readiness Failed

### Symptoms

```
Readiness probe failed
503
```

### Diagnosis

```
kubectl describe pod
kubectl logs
kubectl get endpoints
```

Verified

```
/ready
```

endpoint.

### RCA

Redis connectivity issue.

---

# Issue 13

## Redis Connection Failed

### Diagnosis

```
redis-cli
```

```
PONG
```

Eventually verified connectivity.

### RCA

Redis endpoint configuration.

TLS.

Security Groups.

Application configuration.

### Fix

Corrected

```
REDIS_HOST
REDIS_PORT
```

Configured TLS support.

Verified connectivity.

---

# Issue 14

## Image Updates Not Reflected

### RCA

Using same image tag.

```
1.0
```

Kubernetes reused cached image.

### Fix

Recommended

```
1.1
1.2
2.0
```

or

```
imagePullPolicy: Always
```

---

# Useful Troubleshooting Commands

## Pods

```bash
kubectl get pods -A
kubectl describe pod
kubectl logs
kubectl logs --previous
```

---

## Deployments

```bash
kubectl get deploy
kubectl rollout restart
kubectl rollout status
```

---

## Services

```bash
kubectl get svc
kubectl get endpoints
```

---

## Ingress

```bash
kubectl get ingress
kubectl describe ingress
```

---

## ConfigMaps

```bash
kubectl get configmap
kubectl edit configmap
```

---

## Secrets

```bash
kubectl get secret
```

---

## Helm

```bash
helm list -A
helm status
helm uninstall
helm upgrade
```

---

## AWS

```bash
aws sts get-caller-identity

aws ecr describe-repositories

aws eks update-kubeconfig

aws ec2 describe-instances
```

---

# Root Cause Analysis Summary

| Issue | Root Cause |
|---------|------------|
| Docker Build | Wrong build context |
| ImagePullBackOff | ARM image deployed to AMD64 nodes |
| InvalidImageName | Placeholder not replaced |
| Namespace Missing | Namespace never created |
| Helm Release Exists | Previous failed release |
| ALB Controller | Missing ServiceAccount / IRSA |
| ALB AccessDenied | IAM Policy missing |
| Orders Crash | Placeholder RDS endpoint |
| PostgreSQL | SSL disabled |
| Cart | Redis configuration |
| Web Missing | Deployment not included |
| Redis | TLS / Endpoint configuration |

---

# Key Learnings

## AWS

- EKS node architecture matters.
- IRSA is preferred over node IAM roles.
- ALB Controller requires IAM permissions.
- RDS requires SSL connections.
- ElastiCache Serverless uses TLS.
- Security Groups are critical for service communication.

---

## Kubernetes

- Readiness and Liveness probes behave differently.
- Pods can be Running but not Ready.
- Services only route traffic to Ready Pods.
- Endpoint objects help diagnose readiness issues.
- ReplicaSets help understand rolling updates.
- ImagePullBackOff and CrashLoopBackOff require different troubleshooting approaches.

---

## Docker

- Build images for the correct CPU architecture.
- Use unique image tags.
- Multi-stage builds reduce image size.
- Distroless images improve security.

---

## Helm

- Separate environment-specific values.
- Validate rendered templates before deployment.
- Use `helm upgrade --install`.

---

## Production Best Practices

- Store secrets in AWS Secrets Manager.
- Use IRSA for AWS API access.
- Use immutable image tags.
- Enable TLS everywhere.
- Monitor health checks.
- Validate application readiness before exposing traffic.

---

# Final Outcome

Successfully deployed a production-style microservices application on Amazon EKS using:

- Kubernetes
- Helm
- Amazon ECR
- Amazon RDS PostgreSQL
- Amazon ElastiCache Redis
- AWS Load Balancer Controller
- Amazon ALB
- IAM Roles for Service Accounts (IRSA)

All services became healthy and accessible through the AWS Application Load Balancer.
