#!/usr/bin/env bash
# Build FinLedger images and load into KIND. Requires Maven-built jars (mvn package
# runs INSIDE the Docker multi-stage build, so just docker build is enough).
set -euo pipefail
CLUSTER=${1:-finledger}
cd "$(dirname "$0")/../.."
for svc in transaction-api fraud-worker reconciliation; do
  echo "==> building finledger/$svc:1.0"
  docker build -t "finledger/$svc:1.0" "services/$svc"
  kind load docker-image "finledger/$svc:1.0" --name "$CLUSTER"
done
echo "All FinLedger images built and loaded into KIND '$CLUSTER'."
