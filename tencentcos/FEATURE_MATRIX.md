# 腾讯云 COS 驱动功能矩阵

> 代码依据：`tencentcos.go`、`config.go`、`register.go`（objectx/tencentcos）
> 更新日期：2026-09-09 | 驱动注册名：`tencentcos` | 客户端库：`github.com/tencentyun/cos-go-sdk-v5`

## 1. 基础命令矩阵（BucketClient 接口）

| 接口方法 | 底层 API | 状态 | 行为说明 |
|---|---|---|---|
| `PutObject` | `cos.Client.Object.Put()` | ✅ 支持 | 支持分块上传，大文件自动分段 |
| `GetObject` | `cos.Client.Object.Get()` | ✅ 支持 | 返回 io.ReadCloser + response |
| `DeleteObject` | `cos.Client.Object.Delete()` | ✅ 支持 | — |
| `ListObjects` | `cos.Client.Bucket.Get()` | ✅ 支持 | 支持 prefix 过滤和分页（MaxKeys） |
| `PresignedURL` | `cos.Client.Object.SignURL()` | ✅ 支持 | 生成签名下载 URL |
| `Exists` | `cos.Client.Object.Head()` | ✅ 支持 | Head 不报错即存在 |
| `HealthCheck` | `cos.Client.Bucket.Get()` | ✅ 支持 | — |
| `Close` | — | ✅ 支持 | 空实现 |
| `Driver` | 返回 `"tencentcos"` | ✅ 支持 | — |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 腾讯云 COS（公有云） | ✅ 支持 | 标准 SaaS 接入 |
| 多 Region 部署 | ✅ 支持 | 每个 Region 独立 Endpoint，驱动支持多实例 |
| 腾讯云私有化（TCE） | ⚠️ 部分 | TCE 兼容 S3 协议，但签名略有差异 |
| TLS | ✅ 支持 | 默认 HTTPS |
| 签名认证（STS临时密钥） | ✅ 支持 | 支持长期 AK/SK 和临时 STS 密钥 |
| 其他 S3 兼容服务 | ❌ 不支持 | 腾讯云使用自有签名，非标准 S3 兼容 |

## 3. 配置字段生效状态（objectx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Endpoint` | ✅ 生效 | COS 端点（如 `https://myapp-1250000000.cos.ap-guangzhou.myqcloud.com`） |
| `AccessKey` | ✅ 生效 | 腾讯云 SecretId |
| `SecretKey` | ✅ 生效 | 腾讯云 SecretKey |
| `Bucket` | ✅ 生效 | 存储桶名称（含 appid 后缀） |
| `UseSSL` | ✅ 生效 | — |
| `TLSEnabled` | ✅ 生效 | — |
| `MaxUploadSize` | ✅ 生效 | 单对象最大 5GB |
| `PresignedExpiry` | ✅ 生效 | 默认 1h，最长 7 天 |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Multipart Upload | ✅ 支持 | 原生支持分块上传 |
| Copy Object | ✅ 支持 | 服务端复制 |
| 图片处理 | ⚠️ 部分 | COS 原生图片处理，驱动未封装 URL 参数 |
| 生命周期规则 | ⚠️ 部分 | SDK 支持，驱动未封装 |
| 分片拷贝 | ⚠️ 部分 | SDK 支持，驱动未封装 |
| 数据万象（CI）集成 | ❌ 不支持 | 需业务侧调用 COS CI API |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 图片/视频存储 | ✅ 完整支持 | COS 原生存储 + 图片处理（需额外配置） |
| 静态资源 CDN 源站 | ✅ 完整支持 | COS 配合 CDN 加速 |
| 文件分享链接 | ✅ 完整支持 | PresignedURL |
| 大文件上传 | ✅ 完整支持 | 分块上传 |
| 腾讯云生态集成 | ✅ 优势场景 | 与 CVM/CLS/CDB 等原生集成 |
| 临时密钥上传 | ✅ 支持 | STS 临时凭据，适合前端直传 |
| 数据湖 | ❌ 不支持 | 非 S3 兼容 |
| 多 Region 灾备 | ⚠️ 部分 | 需业务侧跨 Region 复制 |
| 健康探针 | ✅ 支持 | Bucket Get 检查 |
| 链路追踪 | ✅ 支持 | `object.tencentcos.*` span |

## 6. 已知限制与风险

1. **Bucket 名称必须包含 AppID**：COS bucket 命名规范为 `{BucketName}-{AppID}`，创建 bucket 时需遵守，否则请求失败。
2. **非 S3 兼容**：腾讯云 COS 使用自有签名协议，不可直接替换为 S3 客户端。
3. **签名最长 7 天**：PresignedURL 最长有效期 7 天，长期分享链接需业务侧定期刷新。
4. **最大 5GB 单对象**：超过 5GB 需业务侧自行分片（SDK 支持但未封装进 BucketClient 接口）。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/objectx"
    _ "github.com/gospacex/objectx/tencentcos" // 注册驱动
)

client, _ := objectx.New(&objectx.Config{
    Name: "default", Driver: "tencentcos",
    Endpoint: "https://myapp-1250000000.cos.ap-guangzhou.myqcloud.com",
    AccessKey: "YOUR_TENCENT_SECRET_ID", SecretKey: "YOUR_TENCENT_SECRET_KEY",
    Bucket: "myapp-bucket-1250000000", UseSSL: true,
})
// 注意：Bucket 名称须包含 AppID
```
