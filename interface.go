// objectx: 对象存储统一接口，支持 minio / s3 等驱动
package objectx

import (
	"context"
	"fmt"
	"io"
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

var creators = map[string]func(*Config) (BucketClient, error){}

// RegisterCreator 注册存储驱动创建函数
func RegisterCreator(name string, creator func(*Config) (BucketClient, error)) {
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

	creator, ok := creators[cfg.Driver]
	if !ok {
		return nil, fmt.Errorf("objectx: driver %q not registered; make sure to import the driver package", cfg.Driver)
	}

	client, err := creator(cfg)
	if err != nil {
		return nil, fmt.Errorf("objectx: create client %q: %w", cfg.Driver, err)
	}
	return client, nil
}
