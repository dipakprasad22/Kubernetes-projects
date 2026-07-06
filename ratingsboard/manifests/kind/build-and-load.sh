#!/usr/bin/env bash
set -euo pipefail
CLUSTER=${1:-ratingsboard}
cd "$(dirname "$0")/../.."
docker build -t ratingsboard/reporting-api:1.0 services/reporting-api
docker build -t ratingsboard/rating-calc:1.0 services/rating-calc
docker build -t ratingsboard/dashboard:1.0 dashboard
for img in reporting-api rating-calc dashboard; do
  kind load docker-image ratingsboard/$img:1.0 --name "$CLUSTER"
done
echo "All RatingsBoard images built and loaded into KIND '$CLUSTER'."
