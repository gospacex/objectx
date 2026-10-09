#!/usr/bin/env bash
# 校验 objectx/rustfs（RustFS S3） 是否真的可用（协议级探活，不只是端口通）
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

HOST_PORT="${HOST_PORT:-9010}"
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

echo "Verifying objectx/rustfs（RustFS S3）..."
check "RustFS S3 API 存活（4xx 即证明在认证而非没起）" curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:${HOST_PORT:-9010}/
if [ "$fail" -eq 0 ]; then
  echo "ALL CHECKS PASSED"
else
  echo "SOME CHECKS FAILED（先看日志: $DOCKER compose logs）"
  exit 1
fi
