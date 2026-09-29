# MinIO 驱动功能矩阵

> 代码依据：`minio.go`、`config.go`、`register.go`（objectx/minio）
> 更新日期：2026-09-09 | 驱动注册名：`minio` | 客户端库：`github.com/minio/minio-go/v7`

## 1. 基础命令矩阵（BucketClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `PutObject(ctx, bucket, key, data, size, opts...)` | `client.PutObject()` | ✅ 支持 | 支持 PartSize 分片上传，大文件自动分段 |
| `GetObject(ctx, bucket, key)` | `client.GetObject()` | ✅ 支持 | 返回 io.ReadCloser + Object 元信息 |
| `DeleteObject(ctx, bucket, key)` | `client.RemoveObject()` | ✅ 支持 | — |
| `ListObjects(ctx, bucket, prefix, recursive)` | `client.ListObjects()` | ✅ 支持 | 支持前缀过滤和递归遍历 |
| `PresignedURL(ctx, bucket, key, expiry)` | `client.PresignedGetObject()` | ✅ 支持 | 生成预签名下载 URL |
| `Exists(ctx, bucket, key)` | `client.StatObject()` + error check | ✅ 支持 | Stat 不返回错误即存在 |
| `HealthCheck()` | `client.BucketExists()` | ✅ 支持 | 检查 bucket 可达性 |
| `Close()` | `client.CloseIdleConnections()` | ✅ 支持 | — |
| `Driver()` | 返回 `"minio"` | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `object.minio.{OPERATION}` |
| `db.system` | `minio` |
| `cloud.provider` | `other`（S3 兼容） |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单机 MinIO | ✅ 支持 | `cfg.Endpoint` 直连 |
| MinIO 集群（分布式） | ✅ 支持 | `cfg.Endpoints` 多地址，客户端自动分片 |
| AWS S3 | ✅ 支持 | S3 完全兼容，替换 Endpoint 即可 |
| 阿里云 OSS（S3 兼容） | ✅ 支持 | 使用 OSS S3 兼容端点 |
| TLS | ✅ 支持 | `cfg.TLSEnabled` → `http.Transport.TLSConfig` |
| AWS Sig V4 | ✅ 支持 | 原生 V4 签名，兼容所有 S3 体系对象存储 |
| 桶策略（Bucket Policy） | ⚠️ 部分 | 驱动提供 PutBucketPolicy API，但未封装 |

## 3. 配置字段生效状态（objectx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Endpoint` | ✅ 生效 | MinIO/S3 端点地址 |
| `AccessKey` / `SecretKey` | ✅ 生效 | AWS Sig V4 签名凭据 |
| `Bucket` | ✅ 生效 | 默认操作桶名 |
| `UseSSL` | ✅ 生效 | HTTP vs HTTPS |
| `TLSEnabled` | ✅ 生效 | — |
| `MaxUploadSize` | ✅ 生效 | 0 表示无限制，超出返回错误 |
| `PresignedExpiry` | ✅ 生效 | Presigned URL 默认过期时间，默认 1h |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Multipart Upload（大文件） | ✅ 支持 | `PutObject` 自动分片，也可手动 `NewUpload` |
| Copy Object | ✅ 支持 | `client.CopyObject()`，服务端复制 |
| GetObjectAttributes | ✅ 支持 | `client.GetObjectAttributes()` |
| SelectObjectContent | ✅ 支持 | S3 Select 查询对象内容 |
| Lifecycle 配置 | ⚠️ 部分 | 驱动未封装 Lifecycle API |
| Bucket Policy | ⚠️ 部分 | 驱动未封装 PutBucketPolicy |
| Versioning | ⚠️ 部分 | 驱动未封装版本控制 API |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图片/视频存储 | ✅ 完整支持 | PutObject + PresignedURL 下载 |
| 静态资源 CDN 源站 | ✅ 完整支持 | 大文件分片上传 + 预签名 URL |
| 备份归档 | ✅ 完整支持 | 持久化存储，MinIO 自身高可用 |
| 数据湖（对象存储 + 计算分离） | ✅ 完整支持 | S3 兼容，Spark/Flink 可直接读取 |
| 文件分享链接 | ✅ 完整支持 | PresignedURL 生成临时下载链接 |
| 大文件上传（>100MB） | ✅ 完整支持 | 自动 multipart 分片 |
| 多版本文件管理 | ⚠️ 部分 | MinIO 支持版本控制，驱动未封装 |
| 跨地域复制 | ⚠️ 部分 | MinIO 支持 Replication，驱动未封装 |
| 热/冷数据分层 | ❌ 不支持 | 需业务侧通过不同 bucket 实现 |
| 健康探针 | ✅ 支持 | BucketExists 检查 |
| 链路追踪 | ✅ 支持 | `object.minio.*` span |

## 6. 已知限制与风险

1. **不存在桶自动创建**：`PutObject` 前需手动调用 `MakeBucket`，否则返回 `NoSuchBucket`。
2. **MaxUploadSize 仅客户端校验**：超出限制在客户端拒绝，服务端无此限制感知。
3. **ListObjects 分页**：大 bucket 下 ListObjects 默认每批 1000 条，需业务侧自行翻页。
4. **PresignedURL 过期后无效**：过期 URL 无法恢复，需重新生成。
5. **并发 PutObject 限制**：MinIO 客户端默认连接池有限，高并发上传需调大 `PoolSize`。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/objectx"
    _ "github.com/gospacex/objectx/minio" // 注册驱动
)

client, _ := objectx.New(&objectx.Config{
    Name: "default", Driver: "minio",
    Endpoint: "127.0.0.1:9000",
    AccessKey: "minioadmin", SecretKey: "minioadmin123",
    Bucket: "my-bucket", UseSSL: false,
})
```
