#!/usr/bin/env bash
# 校验 objectx/minio（MinIO S3） 是否真的可用（协议级探活，不只是端口通）
set -uo pipefail
cd "$(dirname "$0")/.."

DOCKER="$(command -v docker || true)"
if [ -z "$DOCKER" ] && [ -x /usr/local/bin/docker ]; then
  DOCKER=/usr/local/bin/docker
fi
if [ -z "$DOCKER" ]; then
  echo "ERROR: 未找到 docker CLI。"
  exit 1
fi

HOST_PORT="${HOST_PORT:-9000}"
fail=0

check() {
  local desc="$1"; shift
  if out="$("$@" 2>&1)"; then
    echo "  OK   ${desc}"
    [ -n "${VERBOSE:-}" ] && [ -n "$out" ] && echo "       $out"
  else
    echo "  FAIL ${desc}"
    [ -n "$out" ] && echo "       $out"
    fail=1
  fi
}

echo "Verifying objectx/minio（MinIO S3）..."
check "MinIO liveness" curl -sf http://127.0.0.1:9000/minio/health/live
check "MinIO readiness" curl -sf http://127.0.0.1:9000/minio/health/ready
if [ "$fail" -eq 0 ]; then
  echo "ALL CHECKS PASSED"
else
  echo "SOME CHECKS FAILED（先看日志: $DOCKER compose logs）"
  exit 1
fi
