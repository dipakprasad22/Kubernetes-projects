# Runbook: PanelPulse – Kafka Startup Failure & Aggregator Job Failure

## Incident Summary

During deployment of the PanelPulse application on a local KIND Kubernetes cluster, the Kafka StatefulSet repeatedly crashed, preventing the event pipeline from functioning. After Kafka was fixed, the Aggregator CronJob continued to fail because the `processed_events` table was missing from PostgreSQL.

---

# Symptoms

## Kafka

```
kafka-0   Error   CrashLoopBackOff
```

Kafka logs:

```
Missing required configuration `zookeeper.connect`
```

---

## Aggregator

```
aggregator-xxxxx   Error
```

Logs:

```
ERROR: relation "processed_events" does not exist
```

---

## Impact

- Collector accepted ingest requests.
- Kafka broker was unavailable.
- Processor could not consume events.
- Aggregator CronJob failed.
- `measurement_aggregates` was never populated.
- Downstream RatingsBoard could not calculate ratings.

---

# Architecture

```
Collector
     │
     ▼
Kafka
     │
     ▼
Processor
     │
     ▼
processed_events
     │
     ▼
Aggregator CronJob
     │
     ▼
measurement_aggregates
```

---

# Troubleshooting Steps

---

## Step 1 – Verify Pod Status

```bash
kubectl -n panelpulse get pods
```

Purpose

- Check which workload is failing.
- Verify CrashLoopBackOff/Error state.

Observed

```
kafka-0
aggregator-xxxxx
```

---

## Step 2 – Inspect Kafka Logs

```bash
kubectl -n panelpulse logs kafka-0
```

Observed

```
Missing required configuration zookeeper.connect
```

Finding

Kafka was starting in ZooKeeper mode instead of KRaft mode.

---

## Step 3 – Inspect Previous Logs

```bash
kubectl -n panelpulse logs kafka-0 --previous
```

Purpose

Review previous container crash.

---

## Step 4 – Describe Kafka Pod

```bash
kubectl -n panelpulse describe pod kafka-0
```

Purpose

Verify

- Restart count
- Events
- Exit code
- Mounted volumes
- Environment variables

Observed

```
Exit Code: 1
Restart Count: 4
```

---

## Step 5 – Inspect StatefulSet

```bash
kubectl -n panelpulse describe statefulset kafka
```

and

```bash
kubectl -n panelpulse get statefulset kafka -o yaml
```

Purpose

Verify deployed environment variables.

Finding

Manifest still contained old Kafka configuration.

---

## Step 6 – Correct Kafka Configuration

Updated StatefulSet to use proper KRaft configuration.

Added

```
CLUSTER_ID
KAFKA_CFG_NODE_ID
KAFKA_CFG_PROCESS_ROLES
KAFKA_CFG_CONTROLLER_QUORUM_VOTERS
KAFKA_CFG_LISTENERS
KAFKA_CFG_ADVERTISED_LISTENERS
KAFKA_CFG_CONTROLLER_LISTENER_NAMES
KAFKA_CFG_INTER_BROKER_LISTENER_NAME
KAFKA_CFG_LISTENER_SECURITY_PROTOCOL_MAP
KAFKA_CFG_LOG_DIRS
```

---

## Step 7 – Recreate Kafka

```bash
kubectl apply -f kafka.yaml
```

If necessary

```bash
kubectl delete pod kafka-0 -n panelpulse
```

Verification

```bash
kubectl get pods -n panelpulse
```

Result

```
kafka-0   Running
```

---

# Aggregator Troubleshooting

---

## Step 8 – Inspect Aggregator Logs

```bash
kubectl logs job/aggregator-29723715 -n panelpulse
```

Observed

```
ERROR:

relation processed_events does not exist
```

---

## Step 9 – Verify Database Tables

Connect to PostgreSQL

```bash
kubectl exec -it results-db-0 -n panelpulse -- \
psql -U panelpulse -d panelpulse
```

List tables

```sql
\dt
```

Observed

```
measurement_aggregates
```

Missing

```
processed_events
```

---

## Step 10 – Search Repository

Search for DDL

```bash
grep -R "processed_events" .
```

Search for table creation

```bash
grep -R "CREATE TABLE.*processed_events" .
```

Finding

No SQL or migration created the table.

---

## Step 11 – Inspect Processor

Search JPA entities

```bash
find services/processor -name "*.java" | xargs grep "@Entity"
```

Result

No entities.

---

Check Hibernate

```bash
grep -R "ddl-auto" .
```

Observed

```
spring.jpa.hibernate.ddl-auto=update
```

Finding

Hibernate cannot create tables because there are no entities.

---

## Step 12 – Inspect Processor Logs

```bash
kubectl logs deploy/processor -n panelpulse
```

Finding

Processor started successfully.

No database initialization occurred.

---

# Root Cause

Processor never creates

```
processed_events
```

because

- no Flyway migration
- no Liquibase migration
- no schema.sql
- no Hibernate Entity

Aggregator expects the table to exist.

---

# Temporary Resolution

Created table manually.

Example

```sql
CREATE TABLE processed_events (
    id BIGSERIAL PRIMARY KEY,
    panelist_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    daypart TEXT NOT NULL,
    duration_sec INT NOT NULL,
    processed_at TIMESTAMPTZ DEFAULT now()
);
```

Inserted sample records.

---

## Step 13 – Execute Aggregator Manually

Delete previous job

```bash
kubectl delete job aggregator-now -n panelpulse --ignore-not-found
```

Create job

```bash
kubectl create job \
--from=cronjob/aggregator \
aggregator-now \
-n panelpulse
```

View logs

```bash
kubectl logs job/aggregator-now -n panelpulse
```

Observed

```
aggregation complete:
3 channel/daypart aggregates updated
```

Job completed successfully.

---

# Collector Verification

---

## Verify Health Endpoint

```bash
kubectl port-forward svc/collector 8080:80 -n panelpulse
```

```bash
curl http://localhost:8080/health
```

Response

```
{"status":"ok"}
```

---

## Verify Readiness

```bash
curl http://localhost:8080/ready
```

Response

```
{"status":"ready"}
```

---

## Verify Metrics

```bash
curl http://localhost:8080/metrics
```

Response

```
panelpulse_events_accepted_total 1
panelpulse_events_rejected_total 0
```

---

## Verify Collector Logs

```bash
kubectl logs deploy/collector -n panelpulse
```

Observed

```
produce topic=exposure-events
metrics accepted=1 rejected=0
```

---

# Ingress Verification

List ingress

```bash
kubectl get ingress -n panelpulse
```

Describe ingress

```bash
kubectl describe ingress panelpulse -n panelpulse
```

Observed

```
/ingest → collector
```

Finding

No rule existed for

```
/
```

Therefore

```
http://localhost
```

returns

```
404 Not Found
```

Expected behavior.

---

# Root Cause Analysis (RCA)

## Root Cause 1

Kafka StatefulSet used incomplete KRaft configuration.

The Apache Kafka image started in ZooKeeper mode because required KRaft environment variables were missing.

Impact

- Kafka never started.
- Event pipeline unavailable.

Resolution

Updated StatefulSet with proper KRaft configuration.

---

## Root Cause 2

Aggregator expected

```
processed_events
```

to exist.

The Processor application never created this table because the project contains no database initialization mechanism.

Impact

Aggregator failed with

```
relation processed_events does not exist
```

Resolution

Created table manually and inserted sample data.

---

## Root Cause 3

Ingress only exposed

```
/ingest
```

Browsing

```
http://localhost
```

returned

```
404 Not Found
```

This is expected because no route exists for `/`.

---

# Lessons Learned

- Always verify StatefulSet environment variables after applying changes.
- Validate database schema before running batch jobs.
- Add Flyway or Liquibase migrations for automatic schema creation.
- Use a real Kafka producer instead of the logging stub.
- Validate Ingress routes before testing through a browser.
- Prefer port-forward or an in-cluster debug pod when testing backend APIs.

---

# Preventive Actions

- Introduce Flyway migrations for all database objects.
- Replace Collector logProducer with a real Kafka producer.
- Implement Kafka consumer in Processor to populate `processed_events`.
- Add startup readiness checks for database schema.
- Add integration tests validating:
  - Kafka startup
  - Processor persistence
  - Aggregator execution
  - End-to-end event flow
- Add CI/CD smoke tests to verify the complete ingest → aggregate pipeline before deployment.

---

# Final Status

| Component | Status |
|-----------|--------|
| Kafka | ✅ Healthy |
| PostgreSQL | ✅ Healthy |
| Collector | ✅ Healthy |
| Processor | ✅ Running |
| Aggregator CronJob | ✅ Successful |
| measurement_aggregates | ✅ Populated |
| Collector Metrics | ✅ Available |
| Ingress | ✅ Working as designed |
| End-to-End Pipeline | ⚠ Partially Complete (Collector still uses logging stub instead of a real Kafka producer) |