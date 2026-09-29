# objectx 多驱动功能矩阵

## 驱动总览

| 驱动 | 包路径 | SDK 依赖 | S3 兼容 | 认证方式 | 适用场景 |
|------|--------|----------|---------|----------|----------|
| minio | `objectx/minio` | `minio-go/v7` | ✅ 原生 | AK/SK | 本地私有部署 |
| rustfs | `objectx/rustfs` | 复用 `minio-go/v7` | ✅ 完全兼容 | AK/SK | 本地高性能替代 MinIO |
| qiniu | `objectx/qiniu` | `qiniu/go-sdk/v7` | ❌ | AK/SK + Token | 七牛云生态 |
| huaweiobs | `objectx/huaweiobs` | `huaweicloud-sdk-go-obs` | ❌ | AK/SK HMAC | 华为云生态 |
| tencentcos | `objectx/tencentcos` | `cos-go-sdk-v5` | ❌ | SecretID/SecretKey | 腾讯云生态 |
| aliyunoss | `objectx/aliyunoss` | `alibabacloud-oss-go-sdk-v2` | ❌ | AK/SK HMAC | 阿里云生态 |

## BucketClient 接口功能覆盖

| 方法 | minio | rustfs | qiniu | huaweiobs | tencentcos | aliyunoss |
|------|-------|--------|-------|-----------|------------|-----------|
| PutObject | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| GetObject | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| DeleteObject | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| ListObjects | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| PresignedURL | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| Exists | ✅ | ✅ | ⚠️ HEAD | ✅ | ✅ | ✅ |
| HealthCheck | ✅ noop | ✅ noop | ✅ noop | ✅ noop | ✅ noop | ✅ noop |
| Close | ✅ noop | ✅ noop | ✅ noop | ✅ noop | ✅ noop | ✅ noop |
| Driver | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| OTel tracing | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |

> ⚠️ 七牛 Kodo 无原生 `Exists` API，通过 HEAD 请求模拟

## SDK 版本与依赖

| SDK | 最新版 | Go 最低版本 | GitHub |
|-----|--------|-------------|--------|
| minio-go/v7 | v7.0.84 | Go 1.22 | github.com/minio/minio-go/v7 |
| qiniu/go-sdk/v7 | v7.x | Go 1.10+ | github.com/qiniu/go-sdk |
| huaweicloud-sdk-go-obs | v3.x | Go 1.17+ | github.com/huaweicloud/huaweicloud-sdk-go-obs |
| cos-go-sdk-v5 | v0.7.x | Go 1.13+ | github.com/tencentyun/cos-go-sdk-v5 |
| alibabacloud-oss-go-sdk-v2 | v2.x | Go 1.18+ | github.com/aliyun/alibabacloud-oss-go-sdk-v2 |

## 认证差异

| 驱动 | Endpoint 格式 | 认证头 | 签名算法 |
|------|--------------|--------|----------|
| minio/rustfs | `host:port` | Authorization: AWS4-HMAC-SHA256 | AWS Signature V4 |
| qiniu | 上传: `upload.qiniup.com` / 下载: `<bucket>.qiniucdn.com` | Authorization: QBox <token> | Qiniu HMAC-SHA1 |
| huaweiobs | `obs.<region>.myhuaweicloud.com` | Authorization: OBS <signature> | HMAC-SHA256 (OBS 变体) |
| tencentcos | `<bucket>.cos.<region>.myqcloud.com` | Authorization: q-sign-algo=sha1 | HMAC-SHA1 |
| aliyunoss | `<bucket>.<endpoint>` | Authorization: OSS <signature> | HMAC-SHA1 |

## 用例矩阵

### 基础 CRUD 用例

| 用例 | 描述 | 优先级 |
|------|------|--------|
| put-get-delete | 上传 → 下载验证内容 → 删除 | P0 |
| put-with-metadata | 上传带自定义 metadata → 读取验证 | P0 |
| put-content-type | 指定 ContentType 上传 → 验证 | P0 |
| exists-true | 上传后检查 Exists → true | P0 |
| exists-false | 未上传时检查 Exists → false | P0 |
| delete-nonexistent | 删除不存在的 key → 无错误 | P1 |
| get-nonexistent | 下载不存在的 key → 返回错误 | P1 |

### ListObjects 用例

| 用例 | 描述 | 优先级 |
|------|------|--------|
| list-prefix | 按 prefix 过滤列出 | P0 |
| list-recursive | 递归列出所有子目录 | P0 |
| list-empty | 空 bucket 列出 → 空列表 | P1 |
| list-non-recursive | 非递归列出只返回一层 | P1 |

### PresignedURL 用例

| 用例 | 描述 | 优先级 |
|------|------|--------|
| presigned-get | 生成下载预签名 URL → HTTP 200 | P0 |
| presigned-put | 生成上传预签名 URL → HTTP 可用 | P1 |
| presigned-expiry | 验证过期时间生效 | P1 |

### 错误处理用例

| 用例 | 描述 | 优先级 |
|------|------|--------|
| wrong-endpoint | 错误 endpoint → 返回连接错误 | P0 |
| wrong-credentials | 错误 AK/SK → 返回认证错误 | P0 |
| wrong-bucket | 不存在的 bucket → 返回错误 | P0 |

### 健壮性用例

| 用例 | 描述 | 优先级 |
|------|------|--------|
| large-object | 上传/下载大文件 (>5MB) | P1 |
| concurrent-ops | 并发 Put/Get 不 panic | P1 |
| driver-name | Driver() 返回正确名称 | P0 |

## 测试实现与执行状态

共享功能套件位于 `internal/testutil/suite.go`（`RunSuite` + `LoadConfigFromEnv`）。
各驱动通过 `RunSuite` 跑同一份用例矩阵；构造校验 / 工厂注册为无网络单元测试，恒跑。

| 驱动 | 单元测试 | 功能矩阵 | 集成环境 | 验证时间 |
|------|---------|---------|---------|---------|
| root 工厂 | ✅ | ✅ | —（无网络） | 2026-09-07 通过 |
| middleware (Hertz) | ✅ | ✅ | —（无网络） | 2026-09-07 通过 |
| minio | ✅ | ✅ 11 cases | docker `localhost:9000` | 2026-09-07 通过 |
| rustfs | ✅ | ✅ 11 cases | docker `localhost:9010` | 2026-09-07 通过 |
| qiniu | ✅ | ⏳ 待联调 | `OBJECTX_TEST_QINIU_*` | — |
| huaweiobs | ✅ | ⏳ 待联调 | `OBJECTX_TEST_HUAWEOBS_*` | — |
| tencentcos | ✅ | ⏳ 待联调 | `OBJECTX_TEST_TENCENTCOS_*` | — |
| aliyunoss | ✅ | ⏳ 待联调 | `OBJECTX_TEST_ALIYUNOSS_*` | — |

运行方式：

```bash
# 本地 docker 集成（MinIO + RustFS，含预签名下载验证、-race）
bash scripts/test-docker.sh all        # 或 minio / rustfs

# 云厂商集成（需真实账号，填入环境变量）
OBJECTX_TEST_QINIU_ENDPOINT=... OBJECTX_TEST_QINIU_AK=... OBJECTX_TEST_QINIU_SK=... \
OBJECTX_TEST_QINIU_BUCKET=... go test -v ./qiniu/...         # 其余驱动同理
```

注：功能矩阵中的 `presigned-get` 用例在本地驱动（minio/rustfs）实际下载验证返回 200；
云厂商（受公网/CDN/访问策略影响）仅验证 URL 生成，下载验证按环境可选。
`large-object` 用例默认 1MB，低于矩阵标注的 5MB，通过 `SuiteOptions.LargeSize` 可调。

Middleware 存储语义（2026-09-07 修复）：`Middleware` 原先仅 `c.Set("objectx_client", client)`，
写入的是 Hertz `RequestContext.Keys`；而 `GetClient(ctx)` 走 stdlib `ctx.Value`（Hertz
`RequestContext` 不实现 `context.Context`，引擎传给 handler 的 ctx 是独立存储），导致下游
取不到客户端。修复为 `context.WithValue(ctx, …)` 后经 `c.Next(ctx)` 透传，同时保留 `c.Set`
以支持 `RequestContext.Value` 直接访问。回归用例见 `middleware/hertz_test.go`。
