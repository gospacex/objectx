#!/bin/bash
set -e

SDK="objectx"
DRIVER="${1:-minio}"
REPO_ROOT="/Users/hyx/work/gowork/src/hubx"

echo "=== ${SDK} example: ${DRIVER} ==="
cd "${REPO_ROOT}/${SDK}"

echo "→ Starting docker services..."
docker compose up -d

echo "→ Waiting for healthchecks..."
RETRIES=30
while [ "$RETRIES" -gt 0 ]; do
  if docker inspect --format '{{.State.Health.Status}}' hubx-${DRIVER} 2>/dev/null | grep -q "healthy"; then
    break
  fi
  sleep 2
  RETRIES=$((RETRIES - 1))
done
if [ "$RETRIES" -eq 0 ]; then
  echo "⚠ Healthcheck timeout, proceeding anyway..."
fi

echo "→ Running example..."
cd examples/${DRIVER}
go run .

echo "→ Stopping docker services..."
cd ../../..
docker compose down

echo "✅ ${SDK}/${DRIVER} example passed"