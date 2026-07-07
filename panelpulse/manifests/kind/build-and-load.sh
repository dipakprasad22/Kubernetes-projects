#!/usr/bin/env bash
# Build PanelPulse images and load into KIND.
set -euo pipefail
CLUSTER=${1:-panelpulse}
cd "$(dirname "$0")/../.."
for svc in collector node-agent processor aggregator; do
  echo "==> building panelpulse/$svc:1.0"
  docker build -t "panelpulse/$svc:1.0" "services/$svc"
  kind load docker-image "panelpulse/$svc:1.0" --name "$CLUSTER"
done
echo "All PanelPulse images built and loaded into KIND '$CLUSTER'."
