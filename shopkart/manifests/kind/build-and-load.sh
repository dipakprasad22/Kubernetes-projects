#!/usr/bin/env bash
# Build all ShopKart images and load them into the KIND cluster.
set -euo pipefail
CLUSTER=${1:-shopkart}
cd "$(dirname "$0")/../.."

for svc in gateway catalog cart orders users; do
  echo "==> building shopkart/$svc:1.0"
  docker build -t "shopkart/$svc:1.0" "services/$svc"
  kind load docker-image "shopkart/$svc:1.0" --name "$CLUSTER"
done
echo "==> building shopkart/web:1.0"
docker build -t "shopkart/web:1.0" web
kind load docker-image "shopkart/web:1.0" --name "$CLUSTER"
echo "All images built and loaded into KIND cluster '$CLUSTER'."
