#!/usr/bin/env bash
# objectx - 本地 docker 集成测试脚本
#
# 启动 docker-compose（MinIO + RustFS），等待服务就绪后，
# 对两者的驱动运行完整功能矩阵测试（含预签名下载验证）。
#
# 用法:
#   bash scripts/test-docker.sh          # 全量
#   bash scripts/test-docker.sh minio    # 仅 MinIO
#   bash scripts/test-docker.sh rustfs   # 仅 RustFS
set -euo pipefail

cd "$(dirname "$0")/.."

GO_BIN=${GO_BIN:-/usr/local/bin/go}
# 忽略 shell 中可能错误的 GOROOT 环境变量，固定使用本机 Cell 安装的 Go
GO_ROOT=${OBJECTX_GOROOT:-/usr/local/Cellar/go/1.27.1/libexec}
go_prefix=(env -u GOROOT GOROOT="$GO_ROOT" GOTOOLCHAIN=local HOME="$HOME")

echo "[objectx] docker compose up -d ..."
docker compose up -d

TARGET="${1:-all}"
echo "[objectx] waiting for object-store services ..."

# 等待端口就绪
wait_port() { # name host port
  local name=$1 host=$2 port=$3
  for i in $(seq 1 40); do
    if nc -z "$host" "$port" 2>/dev/null; then
      echo "[objectx]  $name ready on ${host}:${port} (~$((i*3))s)"
      return 0
    fi
    sleep 3
  done
  echo "[objectx]  ERROR: $name not reachable on ${host}:${port} after 120s" >&2
  docker compose logs --tail=30 "$name" || true
  exit 1
}

export OBJECTX_TEST_DOCKER=1
export OBJECTX_TEST_MINIO_AK=minioadmin
export OBJECTX_TEST_MINIO_SK=minioadmin123
export OBJECTX_TEST_RUSTFS_AK=rustfsadmin
export OBJECTX_TEST_RUSTFS_SK=rustfsadmin

BASE_GO=("${go_prefix[@]}" "$GO_BIN" test -race -count=1)

case "$TARGET" in
  minio)
    wait_port minio 127.0.0.1 9000
    "${BASE_GO[@]}" ./minio/...
    ;;
  rustfs)
    wait_port rustfs 127.0.0.1 9010
    "${BASE_GO[@]}" ./rustfs/...
    ;;
  all)
    wait_port minio 127.0.0.1 9000
    wait_port rustfs 127.0.0.1 9010
    "${BASE_GO[@]}" ./minio/... ./rustfs/...
    ;;
  *)
    echo "unknown target: $TARGET (use: all|minio|rustfs)" >&2
    exit 2
    ;;
esac

echo "[objectx] done."