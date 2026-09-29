# 华为云 OBS 驱动功能矩阵

> 代码依据：`huaweiobs.go`、`config.go`、`register.go`（objectx/huaweiobs）
> 更新日期：2026-09-09 | 驱动注册名：`huaweiobs` | 客户端库：`github.com/huaweicloud/huaweicloud-sdk-go-obs`

## 1. 基础命令矩阵（BucketClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `PutObject` | `ObsClient.PutObject()` | ✅ 支持 | 支持分块上传，大文件自动分段 |
| `GetObject` | `ObsClient.GetObject()` | ✅ 支持 | 返回 io.ReadCloser + metadata |
| `DeleteObject` | `ObsClient.DeleteObject()` | ✅ 支持 | — |
| `ListObjects` | `ObsClient.ListObjects()` | ✅ 支持 | 支持 prefix 过滤和分页 |
| `PresignedURL` | `ObsClient.Sign()` | ✅ 支持 | 生成签名下载 URL |
| `Exists` | `ObsClient.HeadObject()` | ✅ 支持 | Head 不报错即存在 |
| `HealthCheck` | `ObsClient.GetBucketLocation()` | ✅ 支持 | — |
| `Close` | `ObsClient.Close()` | ✅ 支持 | — |
| `Driver` | 返回 `"huaweiobs"` | ✅ 支持 | — |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 华为云 OBS（公有云） | ✅ 支持 | 标准 SaaS 接入 |
| 多 Region 部署 | ✅ 支持 | 每个 Region 独立 Endpoint，驱动支持多实例 |
| 华为云 Stack（私有云） | ✅ 支持 | 私有化部署，需配置专属 Endpoint |
| TLS | ✅ 支持 | 默认 HTTPS |
| AK/SK 认证 | ✅ 支持 | 华为云自有签名机制 |
| 其他 S3 兼容服务 | ❌ 不支持 | 华为云 OBS 使用自有签名，非 S3 兼容 |

## 3. 配置字段生效状态（objectx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Endpoint` | ✅ 生效 | OBS 端点（如 `https://obs.cn-north-4.myhuaweicloud.com`） |
| `AccessKey` | ✅ 生效 | 华为云 AK |
| `SecretKey` | ✅ 生效 | 华为云 SK |
| `Bucket` | ✅ 生效 | 存储桶名称 |
| `UseSSL` | ✅ 生效 | — |
| `TLSEnabled` | ✅ 生效 | — |
| `MaxUploadSize` | ✅ 生效 | 单对象最大 5GB（SDK 限制） |
| `PresignedExpiry` | ✅ 生效 | 默认 1h |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Multipart Upload | ✅ 支持 | 原生支持分块上传 |
| Copy Object（服务端复制） | ✅ 支持 | — |
| Lifecycle 规则 | ⚠️ 部分 | SDK 支持，驱动未封装 |
| 跨区域复制 | ❌ 不支持 | 需业务侧多 bucket 实现 |
| 静态网站托管 | ⚠️ 部分 | SDK 支持，驱动未封装 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图片/视频存储 | ✅ 完整支持 | — |
| 华为云生态集成 | ✅ 优势场景 | 与 IAM/ECS/CES 等原生集成 |
| 多 Region 部署 | ✅ 完整支持 | 各 Region 独立 bucket |
| 私有化部署（华为云 Stack） | ✅ 完整支持 | — |
| 文件分享链接 | ✅ 完整支持 | PresignedURL |
| 大文件上传 | ✅ 完整支持 | 分块上传 |
| 数据湖 | ❌ 不支持 | 非 S3 兼容，Spark/Flink 无法直读 |
| 热/冷数据分层 | ⚠️ 部分 | 需配置 OBS 生命周期规则（驱动未封装） |
| 健康探针 | ✅ 支持 | GetBucketLocation |
| 链路追踪 | ✅ 支持 | `object.huaweiobs.*` span |

## 6. 已知限制与风险

1. **非 S3 兼容**：华为云 OBS 使用自有签名协议，不可直接替换为 S3 客户端。
2. **Region 必须匹配**：Endpoint 与 Bucket Region 必须一致，否则返回 400 错误。
3. **MaxUploadSize 5GB 限制**：单对象超过 5GB 需业务侧分片后上传（SDK 支持但接口未暴露阈值）。
4. **签名有效期**：PresignedURL 签名最长 7 天，超出需重新生成。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/objectx"
    _ "github.com/gospacex/objectx/huaweiobs" // 注册驱动
)

client, _ := objectx.New(&objectx.Config{
    Name: "default", Driver: "huaweiobs",
    Endpoint: "https://obs.cn-north-4.myhuaweicloud.com",
    AccessKey: "YOUR_HUAWEI_AK", SecretKey: "YOUR_HUAWEI_SK",
    Bucket: "my-bucket", UseSSL: true,
})
```
