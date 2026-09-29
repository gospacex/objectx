# 七牛云 Kodo 驱动功能矩阵

> 代码依据：`qiniu.go`、`config.go`、`register.go`（objectx/qiniu）
> 更新日期：2026-09-09 | 驱动注册名：`qiniu` | 客户端库：`github.com/qiniu/go-sdk/v7`

## 1. 基础命令矩阵（BucketClient 接口）

| 接口方法 | 底层实现 | 状态 | 行为说明 |
|---|---|---|---|
| `PutObject(ctx, bucket, key, data, size, opts...)` | `putToken.Put()` | ✅ 支持 | 七牛 UploadToken 上传，支持断点续传 |
| `GetObject(ctx, bucket, key)` | `DownloadToken.Download()` | ✅ 支持 | 返回 io.ReadCloser |
| `DeleteObject(ctx, bucket, key)` | `Entry.Delete()` | ✅ 支持 | — |
| `ListObjects(ctx, bucket, prefix, recursive)` | `ListBase()` | ✅ 支持 | 支持 prefix 过滤和 limit 分页 |
| `PresignedURL(ctx, bucket, key, expiry)` | `DownloadToken.MakePrivateURL()` | ✅ 支持 | 生成私有资源下载 URL |
| `Exists(ctx, bucket, key)` | `Stat()`（HEAD 请求） | ✅ 支持 | Stat 不报错即存在；⚠️ 通过 HEAD 模拟，非原生 Exists |
| `HealthCheck()` | `BucketManager.List()` 探测 | ✅ 支持 | — |
| `Close()` | — | ✅ 支持 | 空实现（七牛 SDK 无显式关闭） |
| `Driver()` | 返回 `"qiniu"` | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `object.qiniu.{OPERATION}` |
| `db.system` | `qiniu-kodo` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 七牛云 Kodo（公有云） | ✅ 支持 | 标准 SaaS 接入 |
| 七牛私有云 | ✅ 支持 | 私有化部署 Kodo |
| 多区域（zone） | ✅ 支持 | `cfg.Endpoint` 指定区域端点（华东/华南/华北等） |
| TLS | ✅ 支持 | 默认 HTTPS |
| HMAC-SHA1 认证 | ✅ 支持 | 七牛特有签名机制 |
| 其他 S3 兼容服务 | ❌ 不支持 | 七牛使用自有协议，非 S3 兼容 |

## 3. 配置字段生效状态（objectx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Endpoint` | ✅ 生效 | 七牛上传/下载域名（如 `http://up.qiniup.com`） |
| `AccessKey` | ✅ 生效 | 七牛 Access Key |
| `SecretKey` | ✅ 生效 | 七牛 Secret Key，用于 HMAC-SHA1 签名 |
| `Bucket` | ✅ 生效 | 存储空间名称 |
| `UseSSL` | ✅ 生效 | HTTP/HTTPS 切换 |
| `TLSEnabled` | ✅ 生效 | — |
| `MaxUploadSize` | ✅ 生效 | 七牛单文件最大 5GB（SDK 自动分片） |
| `PresignedExpiry` | ✅ 生效 | 私有 URL 默认过期时间，默认 1h |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| 图片处理（Image Processing） | ⚠️ 部分 | 七牛强大图片处理能力，驱动未封装 URL 参数 |
| 音视频转码（Media Processing） | ❌ 不支持 | — |
| 生命周期规则 | ⚠️ 部分 | 七牛云控制台可配置，驱动未封装 API |
| 跨区域复制 | ❌ 不支持 | — |
| 版本控制 | ❌ 不支持 | 七牛云原生不支持对象版本 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图片/视频存储 | ✅ 完整支持 | 七牛云原生图片处理 + 存储 |
| 静态资源 CDN 源站 | ✅ 完整支持 | 七牛 CDN 加速（需额外配置） |
| 文件分享链接 | ✅ 完整支持 | PresignedURL 私有资源下载 |
| 大文件上传 | ✅ 完整支持 | SDK 自动分片，最大 5GB |
| 备份归档 | ✅ 支持 | — |
| 多区域部署 | ✅ 支持 | 不同区域独立 bucket |
| 图片处理流水线 | ⚠️ 部分 | 七牛图片处理能力强但驱动未封装 URL 参数 |
| 数据湖 | ❌ 不支持 | 七牛非 S3 兼容，Spark/Flink 无法直接读取 |
| 健康探针 | ✅ 支持 | BucketManager.List 检查 |
| 链路追踪 | ✅ 支持 | `object.qiniu.*` span |

## 6. 已知限制与风险

1. **非 S3 兼容**：七牛使用自有协议，无法直接替换 AWS S3 客户端，迁移至其他厂商需切换驱动。
2. **Exists 通过 HEAD 模拟**：Stat 请求有费用，高频 Exists 查询会产生额外 API 调用成本。
3. **HMAC-SHA1 签名**：七牛自有签名机制与 AWS Sig V4 不同，需使用七牛 SDK 而非通用 S3 客户端。
4. **图片处理 URL 未封装**：七牛核心优势是图片处理，但驱动未暴露图片处理参数（缩放/裁剪/水印）。
5. **无版本控制**：七牛云对象存储不支持版本，覆盖写会丢失旧版本，需业务侧自持版本管理。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/objectx"
    _ "github.com/gospacex/objectx/qiniu" // 注册驱动
)

client, _ := objectx.New(&objectx.Config{
    Name: "default", Driver: "qiniu",
    Endpoint: "http://up.qiniup.com",
    AccessKey: "YOUR_QINIU_ACCESS_KEY",
    SecretKey: "YOUR_QINIU_SECRET_KEY",
    Bucket: "my-bucket", UseSSL: false,
})
```
