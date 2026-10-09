// objectx: 对象存储统一接口，支持 minio / s3 等驱动
package objectx

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
)

// Object 对象信息
type Object struct {
	Key      string
	Size     int64
	ETag     string
	LastMod  time.Time
	Metadata map[string]string
}

// BucketClient 存储桶客户端接口
type BucketClient interface {
	// PutObject 上传对象
	PutObject(ctx context.Context, bucket, key string, data io.Reader, size int64, opts ...PutOption) error
	// GetObject 下载对象
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *Object, error)
	// DeleteObject 删除对象
	DeleteObject(ctx context.Context, bucket, key string) error
	// ListObjects 列出对象
	ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]Object, error)
	// PresignedURL 生成预签名 URL
	PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	// Exists 检查对象是否存在
	Exists(ctx context.Context, bucket, key string) (bool, error)
	// HealthCheck 健康检查
	HealthCheck() error
	// Close 关闭连接
	Close() error
	// Driver 返回驱动名称
	Driver() string
}

// BucketManager 可选能力：创建桶。
//
// ⚠️ 这是**可选**接口，不并入 BucketClient —— 并入会让全部 6 个既有驱动
// 立即编译失败（破坏性变更），而四个云厂商驱动的建桶语义（区域、存储类型、
// 权限）各不相同，无法用同一个签名安全表达。
//
// 约定：调用方用类型断言判定支持性，**不支持时必须显式跳过或报错，
// 不得假设"桶已存在"静默继续**。BucketClient 本身没有建桶能力，
// 因此在干净的存储服务上（例如刚 docker compose up 的 rustfs/minio）
// 直接 PutObject 会失败，示例必须先走这一步。
//
// 当前实现该能力的驱动：minio、rustfs（均为 S3 兼容，复用 minio-go）。
// 四个云厂商驱动（aliyunoss/huaweiobs/qiniu/tencentcos）未实现，
// 需要先在控制台手动建桶。
type BucketManager interface {
	// MakeBucket 创建桶；桶已存在时应返回 nil（幂等），不要报 "已存在" 错误。
	MakeBucket(ctx context.Context, bucket string) error
}

// PutOption 上传选项
type PutOption func(*PutOptions)

// PutOptions 上传选项
type PutOptions struct {
	ContentType string
	Metadata    map[string]string
}

// WithContentType 设置内容类型
func WithContentType(ct string) PutOption {
	return func(o *PutOptions) { o.ContentType = ct }
}

// WithMetadata 设置元数据
func WithMetadata(m map[string]string) PutOption {
	return func(o *PutOptions) { o.Metadata = m }
}

// Config 对象存储配置
type Config struct {
	Driver          string
	Name            string
	Endpoint        string
	AccessKey       string
	SecretKey       string
	Bucket          string
	UseSSL          bool
	TLSEnabled      bool
	MaxUploadSize   int64 // bytes, 0 表示无限制
	PresignedExpiry time.Duration
	EnableTrace     bool
}

// FillDefaults 填充默认值
func (c *Config) FillDefaults() {
	if c.PresignedExpiry <= 0 {
		c.PresignedExpiry = time.Hour
	}
	if c.Name == "" {
		c.Name = "default"
	}
}

// ===== Factory =====

// creatorsMu 保护下面的 creators map。
//
// 背景：REVIEW-REPORT.md 第二优先级第 6 条要求所有 factory map 加
// sync.RWMutex，第八节已修复清单只覆盖了 mqx/kvx/searchx，后续又补了
// coordx/lockx/vectorx —— objectx 是最后一个漏网的。
// 未加锁时并发注册 + 并发查找会触发 Go 运行时的
// "fatal error: concurrent map read and map write"：这不是普通数据竞态，
// 而是无法 recover 的崩溃级故障，进程直接挂掉
// （已由 interface_test.go 的 TestFactoryConcurrentAccess 在 -race 下复现）。
var creatorsMu sync.RWMutex

var creators = map[string]func(*Config) (BucketClient, error){}

// RegisterCreator 注册存储驱动创建函数
func RegisterCreator(name string, creator func(*Config) (BucketClient, error)) {
	creatorsMu.Lock()
	defer creatorsMu.Unlock()
	creators[name] = creator
}

// New 创建存储客户端
func New(cfg *Config) (BucketClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("objectx: nil config")
	}
	cfg.FillDefaults()
	// 接入 hubx 统一调用链开关：组件自身配置为主开关，
	// 全局总闸 traceflag.SetEnabled(false) 可一键熔断 objectx。
	cfg.EnableTrace = traceflag.Apply("objectx", cfg.EnableTrace)

	// 只锁 map 查找，creator 放到锁外调用 —— 避免 creator 内部再注册
	// 其它驱动时自死锁，也缩短持锁时间。
	creatorsMu.RLock()
	creator, ok := creators[cfg.Driver]
	creatorsMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("objectx: driver %q not registered; make sure to import the driver package", cfg.Driver)
	}

	client, err := creator(cfg)
	if err != nil {
		return nil, fmt.Errorf("objectx: create client %q: %w", cfg.Driver, err)
	}
	return client, nil
}
