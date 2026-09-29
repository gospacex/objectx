# RustFS 驱动功能矩阵

> 代码依据：`rustfs.go`、`config.go`、`register.go`（objectx/rustfs）
> 更新日期：2026-09-09 | 驱动注册名：`rustfs` | 客户端库：`github.com/minio/minio-go/v7`（S3 兼容）

## 1. 基础命令矩阵（BucketClient 接口）

| 接口方法 | 状态 | 行为说明 |
|---|---|---|
| `PutObject` | ✅ 支持 | S3 兼容，与 minio 驱动完全一致 |
| `GetObject` | ✅ 支持 | — |
| `DeleteObject` | ✅ 支持 | — |
| `ListObjects` | ✅ 支持 | — |
| `PresignedURL` | ✅ 支持 | — |
| `Exists` | ✅ 支持 | 通过 StatObject 实现 |
| `HealthCheck` | ✅ 支持 | BucketExists 检查 |
| `Close` | ✅ 支持 | — |
| `Driver` | ✅ 支持 | 返回 `"rustfs"` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单机 RustFS | ✅ 支持 | `cfg.Endpoint` 直连 |
| RustFS 集群 | ✅ 支持 | S3 完全兼容，多 endpoint 分片 |
| TLS | ✅ 支持 | `cfg.TLSEnabled` |
| AWS Sig V4 | ✅ 支持 | 同 minio 驱动 |
| 其他 S3 兼容服务 | ✅ 支持 | 替换 Endpoint 即可对接 AWS S3、阿里云 OSS 等 |

> RustFS 是 S3 完全兼容的对象存储服务，接口与 minio 驱动保持一致，仅驱动名不同。

## 3. 配置字段生效状态（objectx.Config）

与 objectx/minio 完全一致：`Endpoint` / `AccessKey` / `SecretKey` / `Bucket` / `UseSSL` / `TLSEnabled` / `MaxUploadSize` / `PresignedExpiry` / `EnableTrace` / `Name` / `Driver` 全部生效。

## 4. 扩展命令矩阵

与 objectx/minio 完全一致：Multipart Upload、Copy Object、SelectObjectContent 等均支持。

## 5. 企业级业务场景矩阵

与 objectx/minio 完全一致：图片/视频存储、CDN 源站、备份归档、数据湖、文件分享链接、大文件上传等场景均完整支持。

## 6. 已知限制与风险

与 objectx/minio 完全一致，主要风险：
1. 不存在桶自动创建，需提前 `MakeBucket`。
2. `ListObjects` 每批 1000 条需业务侧翻页。
3. PresignedURL 过期后无效。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/objectx"
    _ "github.com/gospacex/objectx/rustfs" // 注册驱动
)

client, _ := objectx.New(&objectx.Config{
    Name: "default", Driver: "rustfs",
    Endpoint: "127.0.0.1:9000",
    AccessKey: "rustfsadmin", SecretKey: "rustfs123",
    Bucket: "my-bucket", UseSSL: false,
})
```
