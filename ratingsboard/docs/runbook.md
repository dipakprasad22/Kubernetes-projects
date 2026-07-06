# Runbook: Troubleshooting RatingsBoard on Kind Kubernetes

## Overview

This runbook documents the end-to-end troubleshooting performed while deploying **RatingsBoard** on a local **Kind Kubernetes** cluster with **NGINX Ingress**, **PostgreSQL**, and **FastAPI**. It captures the symptoms, diagnostic steps, commands used, root causes, and resolutions.

---

# Environment

| Component  | Value                            |
| ---------- | -------------------------------- |
| Kubernetes | Kind v1.36.1                     |
| Ingress    | NGINX Ingress Controller v1.15.1 |
| Database   | PostgreSQL 16                    |
| API        | FastAPI                          |
| Namespace  | ratingsboard                     |

---

# Issue 1 – Kind Cluster Creation Failed

## Symptom

```text
Bind for 0.0.0.0:80 failed: port is already allocated
```

## Root Cause

Host port **80** was already occupied by another process, preventing the Kind control-plane container from exposing port 80.

---

## Troubleshooting Commands

### Check which process is using port 80

```bash
sudo lsof -i :80
```

or

```bash
sudo netstat -anv | grep '\.80 '
```

or

```bash
docker ps
```

### Stop the conflicting process/container

```bash
docker stop <container-id>
```

or

```bash
sudo apachectl stop
```

or

```bash
brew services stop nginx
```

### Verify port is free

```bash
sudo lsof -i :80
```

### Create the Kind cluster

```bash
kind create cluster --name ratingsboard --config manifests/kind/kind-cluster.yaml
```

---

# Issue 2 – Dashboard Pods Stuck in CreateContainerConfigError

## Symptom

```text
CreateContainerConfigError
```

Dashboard pods never started.

---

## Troubleshooting Commands

### Check pod status

```bash
kubectl -n ratingsboard get pods
```

### Describe the failing pod

```bash
kubectl -n ratingsboard describe pod <dashboard-pod>
```

### Check deployment

```bash
kubectl -n ratingsboard describe deployment dashboard
```

### Verify ConfigMaps

```bash
kubectl -n ratingsboard get configmaps
```

### Verify Secrets

```bash
kubectl -n ratingsboard get secrets
```

### View Events

```bash
kubectl -n ratingsboard get events --sort-by=.metadata.creationTimestamp
```

---

## Error

```text
container has runAsNonRoot and image has non-numeric user (node)
```

---

## Root Cause

The Kubernetes Deployment enforced:

```yaml
runAsNonRoot: true
```

but the Docker image was built with

```dockerfile
USER node
```

Kubernetes cannot verify that the named user is non-root.

---

## Resolution

### Verify image user

```bash
docker image inspect ratingsboard/dashboard:1.0 --format '{{.Config.User}}'
```

Expected

```text
1000
```

Incorrect

```text
node
```

### Update Dockerfile

```dockerfile
USER 1000
```

### Rebuild image

```bash
docker build --no-cache -t ratingsboard/dashboard:1.0 .
```

### Load image into Kind

```bash
kind load docker-image ratingsboard/dashboard:1.0 --name ratingsboard
```

### Restart deployment

```bash
kubectl rollout restart deployment dashboard -n ratingsboard
```

---

# Issue 3 – Rating Calculation Job Could Not Resolve Database Host

## Symptom

```text
could not translate host name
panelpulse-results-db.panelpulse.svc.cluster.local
```

---

## Root Cause

The ConfigMap still contained references from the previous **PanelPulse** project.

---

## Verify ConfigMap

```bash
kubectl -n ratingsboard get configmap ratingsboard-config -o yaml
```

---

## Search repository

```bash
grep -R "panelpulse" .
```

---

## Resolution

Updated

```yaml
SRC_DB_HOST: reports-db
SRC_DB_NAME: ratingsboard
SRC_DB_USER: ratingsboard
```

Apply

```bash
kubectl apply -f manifests/base/02-config.yaml
```

Delete old Job

```bash
kubectl -n ratingsboard delete job calc-now
```

Create new Job

```bash
kubectl create job --from=cronjob/rating-calc calc-now -n ratingsboard
```

View logs

```bash
kubectl -n ratingsboard logs -f job/calc-now
```

---

# Issue 4 – measurement_aggregates Table Missing

## Symptom

```text
relation "measurement_aggregates" does not exist
```

---

## Root Cause

The application expected data produced by the upstream PanelPulse application.

The table never existed in the local PostgreSQL database.

---

## Verify database

```bash
kubectl -n ratingsboard exec -it reports-db-0 -- \
psql -U ratingsboard -d ratingsboard
```

List tables

```sql
\dt
```

---

## Resolution

Create table

```sql
CREATE TABLE measurement_aggregates (
    channel_id VARCHAR(100),
    daypart VARCHAR(50),
    impressions BIGINT,
    viewing_min BIGINT,
    reach BIGINT
);
```

Insert sample data

```sql
INSERT INTO measurement_aggregates
(channel_id, daypart, impressions, viewing_min, reach)
VALUES
('NEWS24','Morning',15000,3200,5200),
('SPORTS','Morning',21000,4800,7600),
('MOVIES','Prime',43000,12000,18000);
```

Re-run CronJob

```bash
kubectl delete job calc-now -n ratingsboard

kubectl create job --from=cronjob/rating-calc calc-now -n ratingsboard
```

Verify reports

```sql
SELECT COUNT(*) FROM reports;
```

---

# Issue 5 – API Returned HTTP 404

## Symptom

```text
GET /ratings/top
404 Not Found
```

---

## Troubleshooting

Port forward

```bash
kubectl -n ratingsboard port-forward svc/reporting-api 8000:8000
```

Test API

```bash
curl "http://localhost:8000/api/ratings/top?limit=10"
```

---

## Root Cause

Ingress rewrite removed `/api`

```
/api/ratings/top
        ↓
/ratings/top
```

FastAPI exposed

```
/api/ratings/top
```

---

## Resolution

Remove

```yaml
nginx.ingress.kubernetes.io/rewrite-target
```

Use Prefix paths

```yaml
- path: /api
  pathType: Prefix

- path: /
  pathType: Prefix
```

---

# Issue 6 – Empty Reply From Server

## Symptom

```text
curl: (52) Empty reply from server
```

---

## Troubleshooting Commands

Describe ingress

```bash
kubectl -n ratingsboard describe ingress ratingsboard
```

Check endpoints

```bash
kubectl -n ratingsboard get endpoints reporting-api
```

Verify ingress controller

```bash
kubectl -n ingress-nginx get pods
```

Inspect deployment

```bash
kubectl -n ingress-nginx get deploy ingress-nginx-controller -o yaml
```

Describe pod

```bash
kubectl -n ingress-nginx describe pod -l app.kubernetes.io/component=controller
```

Check docker port mapping

```bash
docker ps
```

```bash
docker port ratingsboard-control-plane
```

Verify controller locally

```bash
kubectl -n ingress-nginx exec -it deploy/ingress-nginx-controller -- \
curl -I http://127.0.0.1
```

---

## Root Cause

The ingress controller was scheduled on the **worker node**.

```
ratingsboard-worker
```

However, Kind published port 80 only from the **control-plane node**.

```
localhost:80
        ↓
control-plane
        ↓
No ingress controller
```

---

## Resolution

Verify node labels

```bash
kubectl get nodes --show-labels
```

Patch deployment

```bash
kubectl -n ingress-nginx patch deployment ingress-nginx-controller \
--type='merge' \
-p '{
  "spec": {
    "template": {
      "spec": {
        "nodeSelector": {
          "ingress-ready": "true",
          "kubernetes.io/os": "linux"
        }
      }
    }
  }
}'
```

Wait for rollout

```bash
kubectl -n ingress-nginx rollout status deployment ingress-nginx-controller
```

Verify scheduling

```bash
kubectl -n ingress-nginx get pods -o wide
```

Expected

```
ratingsboard-control-plane
```

---

# Verification Commands

## Pods

```bash
kubectl -n ratingsboard get pods
```

## Services

```bash
kubectl -n ratingsboard get svc
```

## Endpoints

```bash
kubectl -n ratingsboard get endpoints
```

## Ingress

```bash
kubectl -n ratingsboard describe ingress ratingsboard
```

## Database Tables

```sql
\dt
```

## Reports Count

```sql
SELECT COUNT(*) FROM reports;
```

## API

```bash
curl "http://localhost:8000/api/ratings/top?limit=10"
```

or

```bash
curl "http://localhost/api/ratings/top?limit=10"
```

---

# Root Cause Analysis (RCA)

| Issue                        | Root Cause                                                      | Resolution                                                             |
| ---------------------------- | --------------------------------------------------------------- | ---------------------------------------------------------------------- |
| Kind cluster creation failed | Host port 80 already in use                                     | Free port 80 or change host port                                       |
| Dashboard pods failed        | Docker image used `USER node` while Pod required `runAsNonRoot` | Rebuild image using numeric UID (`USER 1000`)                          |
| Rating Job failed            | ConfigMap still referenced old PanelPulse database              | Update ConfigMap with RatingsBoard database details                    |
| CronJob skipped processing   | `measurement_aggregates` table absent                           | Create and seed source table                                           |
| API returned 404             | Ingress rewrite stripped `/api` prefix                          | Remove rewrite annotation and use `Prefix` paths                       |
| Empty reply from server      | Ingress controller scheduled on worker instead of control-plane | Schedule ingress controller on `ingress-ready=true` control-plane node |

---

# Lessons Learned

* Always verify the effective Docker image configuration (`docker image inspect`) after rebuilding images.
* Remember that Kind requires `kind load docker-image` after local image rebuilds.
* Validate `ConfigMap` values after project renaming to avoid stale environment variables.
* Test applications directly using `kubectl port-forward` before troubleshooting Ingress.
* Confirm Kubernetes Endpoints before investigating network routing.
* For Kind, ensure the NGINX Ingress controller runs on the control-plane node that exposes the host ports.
* Differentiate infrastructure issues (networking, scheduling, security context) from application issues (missing schema, incorrect routes) to reduce troubleshooting time.
