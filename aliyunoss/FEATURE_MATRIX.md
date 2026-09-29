# 阿里云 OSS 驱动功能矩阵

> 代码依据：`aliyunoss.go`、`config.go`、`register.go`（objectx/aliyunoss）
> 更新日期：2026-09-09 | 驱动注册名：`aliyunoss` | 客户端库：`github.com/aliyun/aliyun-oss-go-sdk`

## 1. 基础命令矩阵（BucketClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `PutObject` | `bucket.PutObject(key, reader)` | ✅ 支持 | 支持分块上传（ResumableUpload） |
| `GetObject` | `bucket.GetObject(key)` | ✅ 支持 | 返回 io.ReadCloser |
| `DeleteObject` | `bucket.DeleteObject(key)` | ✅ 支持 | — |
| `ListObjects` | `bucket.ListObjects()` | ✅ 支持 | 支持 prefix 过滤和分页（MaxKeys） |
| `PresignedURL` | `bucket.SignURL(key, http.MethodGet, expiry)` | ✅ 支持 | 生成签名下载 URL |
| `Exists` | `bucket.IsObjectExist(key)` | ✅ 支持 | — |
| `HealthCheck` | `bucket.HeadBucket()` | ✅ 支持 | — |
| `Close` | — | ✅ 支持 | 空实现（阿里云 SDK 无显式关闭） |
| `Driver` | 返回 `"aliyunoss"` | ✅ 支持 | — |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 阿里云 OSS（公有云） | ✅ 支持 | 标准 SaaS 接入 |
| 多 Region 部署 | ✅ 支持 | 每个 Region 独立 Endpoint，驱动支持多实例 |
| 阿里专有云（Apsara） | ⚠️ 部分 | 私有化部署，Endpoint 需替换 |
| TLS | ✅ 支持 | 默认 HTTPS |
| RAM 子账户 | ✅ 支持 | 支持 RAM Role/STS 临时凭据 |
| 其他 S3 兼容服务 | ❌ 不支持 | 阿里云 OSS 使用自有签名 V4，非标准 S3 |

## 3. 配置字段生效状态（objectx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Endpoint` | ✅ 生效 | OSS 端点（如 `https://oss-cn-hangzhou.aliyuncs.com`） |
| `AccessKey` | ✅ 生效 | 阿里云 AK |
| `SecretKey` | ✅ 生效 | 阿里云 SK |
| `Bucket` | ✅ 生效 | 存储桶名称（不含 region） |
| `UseSSL` | ✅ 生效 | — |
| `TLSEnabled` | ✅ 生效 | — |
| `MaxUploadSize` | ✅ 生效 | 单对象最大 5GB（分片上传可突破） |
| `PresignedExpiry` | ✅ 生效 | 默认 1h，最长 1 小时（v2 SDK 限制） |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Multipart Upload | ✅ 支持 | `ResumableUpload` 原生支持断点续传 |
| Copy Object | ✅ 支持 | `CopyObject` 服务端复制 |
| Lifecycle 规则 | ⚠️ 部分 | SDK 支持，驱动未封装 |
| 图片处理（Image Service） | ⚠️ 部分 | OSS 图片处理能力强，驱动未封装 URL 参数 |
| 跨区域复制 | ❌ 不支持 | 需业务侧双 bucket 实现 |
| 版本控制 | ⚠️ 部分 | OSS 支持版本控制，驱动未封装 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图片/视频存储 | ✅ 完整支持 | OSS 原生存储 + 图片处理（需额外配置） |
| 静态资源 CDN 源站 | ✅ 完整支持 | OSS + CDN 联动 |
| 大文件上传 | ✅ 完整支持 | ResumableUpload 断点续传 |
| 文件分享链接 | ✅ 完整支持 | PresignedURL |
| 阿里云生态集成 | ✅ 优势场景 | 与 ECS/RDS/SLB 等原生集成 |
| 数据湖 | ❌ 不支持 | 非 S3 兼容 |
| 多 Region 灾备 | ⚠️ 部分 | 需业务侧跨 Region 复制 |
| 模板任务/批量操作 | ⚠️ 部分 | OSS 支持 Batch API，驱动未封装 |
| 健康探针 | ✅ 支持 | HeadBucket 检查 |
| 链路追踪 | ✅ 支持 | `object.aliyunoss.*` span |

## 6. 已知限制与风险

1. **PresignedURL 最长 1 小时**：阿里云 v2 SDK 限制签名 URL 最长有效期为 1 小时，超出需重新生成（不同于其他厂商的 7 天）。
2. **非 S3 兼容**：阿里云 OSS 使用自有签名机制，不可直接替换为 S3 客户端。
3. **Endpoint 必须精确到 Region**：`oss-cn-hangzhou.aliyuncs.com` 格式，Region 错误返回 400。
4. **CNAME 绑定限制**：自定义域名需备案且绑定，SDK 不支持直接使用未绑定的 CNAME。
5. **v2 SDK API 差异**：阿里云 OSS Go SDK v2 与早期 v1 API 有差异，部分高级功能（如 Batch、版本控制）未在驱动中封装。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/objectx"
    _ "github.com/gospacex/objectx/aliyunoss" // 注册驱动
)

client, _ := objectx.New(&objectx.Config{
    Name: "default", Driver: "aliyunoss",
    Endpoint: "https://oss-cn-hangzhou.aliyuncs.com",
    AccessKey: "YOUR_ALIYUN_AK", SecretKey: "YOUR_ALIYUN_SK",
    Bucket: "my-bucket", UseSSL: true,
})
```
