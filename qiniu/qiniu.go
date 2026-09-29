// objectx/qiniu: 七牛云 Kodo 对象存储驱动
//
// Config 字段映射：
//   - Endpoint  → 七牛上传域名（如 https://up.qiniup.com 或自定义上传域名）
//   - AccessKey → 七牛 Access Key
//   - SecretKey → 七牛 Secret Key
//   - Bucket    → 存储空间名称
//
// 下载域名说明：
//
//	七牛下载通常走独立域名（如 <bucket>.qiniucdn.com 或自定义域名），
//	与上传域名不同。本驱动在 PresignedURL 和 GetObject 中使用 Endpoint
//	作为下载域名；若下载域名与上传域名不同，请在调用方自行处理。
package qiniu

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	"github.com/gospacex/traceflag"
	"github.com/qiniu/go-sdk/v7/auth/qbox"
	"github.com/qiniu/go-sdk/v7/client"
	"github.com/qiniu/go-sdk/v7/storage"
	"go.opentelemetry.io/otel/attribute"
)

// ClientImpl 七牛云 Kodo 客户端
type ClientImpl struct {
	mac      *qbox.Mac
	cfg      *objectx.Config
	uploader *storage.FormUploader
	tracer   *traceflag.Tracer
	logger   logx.Logger
}

// New 创建七牛云 Kodo 客户端
func New(cfg *objectx.Config) (*ClientImpl, error) {
	if cfg.Driver != "qiniu" {
		return nil, fmt.Errorf("qiniu: expected driver 'qiniu', got %q", cfg.Driver)
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("qiniu: AccessKey and SecretKey are required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("qiniu: Bucket is required")
	}

	mac := qbox.NewMac(cfg.AccessKey, cfg.SecretKey)

	cfgC := storage.Config{
		UseHTTPS: cfg.UseSSL,
	}
	uploader := storage.NewFormUploader(&cfgC)

	c := &ClientImpl{
		mac:      mac,
		cfg:      cfg,
		uploader: uploader,
		tracer:   traceflag.NewTracer(cfg != nil && cfg.EnableTrace, "github.com/gospacex/objectx/qiniu", "objectx"),
		logger:   logx.GetGlobal(),
	}
	c.logger.InfoContext(context.Background(), "objectx/qiniu initialized", logx.String("bucket", cfg.Bucket))
	return c, nil
}

// PutObject 上传对象
func (c *ClientImpl) PutObject(ctx context.Context, bucket, key string, data io.Reader, size int64, opts ...objectx.PutOption) error {
	ctx, span := c.tracer.Start(ctx, "object_store.put")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
		attribute.Int64("object_store.size", size),
	)

	popts := objectx.PutOptions{}
	for _, opt := range opts {
		opt(&popts)
	}

	putPolicy := storage.PutPolicy{
		Scope: fmt.Sprintf("%s:%s", bucket, key),
	}
	token := putPolicy.UploadToken(c.mac)

	putExtra := storage.PutExtra{}
	if popts.ContentType != "" {
		putExtra.MimeType = popts.ContentType
	}
	if len(popts.Metadata) > 0 {
		putExtra.Params = popts.Metadata
	}

	var ret storage.PutRet
	err := c.uploader.Put(ctx, &ret, token, key, data, size, &putExtra)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/qiniu put failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("qiniu: put object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/qiniu put done", logx.String("bucket", bucket), logx.String("key", key), logx.String("ret_key", ret.Key))
	return nil
}

// GetObject 下载对象
//
// 七牛 SDK 不直接提供返回 io.ReadCloser 的下载 API，这里通过生成私有签名
// 下载链接，再用 http.Client 拉取。GetObject 返回的对象元信息（如 ETag、
// LastMod）通常不含在下载响应中，调用方可使用 PresignedURL 做精确控制。
func (c *ClientImpl) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	domain := c.cfg.Endpoint
	if domain == "" {
		return nil, nil, fmt.Errorf("qiniu: get object %s/%s: Endpoint (download domain) is required", bucket, key)
	}

	expiry := c.cfg.PresignedExpiry
	if expiry <= 0 {
		expiry = time.Hour
	}
	dlURL := storage.MakePrivateURL(c.mac, domain, key, time.Now().Add(expiry).Unix())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dlURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("qiniu: get object %s/%s: %w", bucket, key, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("qiniu: get object %s/%s: %w", bucket, key, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("qiniu: get object %s/%s: unexpected status %d", bucket, key, resp.StatusCode)
	}

	obj := &objectx.Object{
		Key:  key,
		Size: resp.ContentLength,
	}
	return resp.Body, obj, nil
}

// DeleteObject 删除对象
func (c *ClientImpl) DeleteObject(ctx context.Context, bucket, key string) error {
	ctx, span := c.tracer.Start(ctx, "object_store.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	bucketManager := storage.NewBucketManager(c.mac, nil)
	err := bucketManager.Delete(bucket, key)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/qiniu delete failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("qiniu: delete object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/qiniu delete done", logx.String("bucket", bucket), logx.String("key", key))
	return nil
}

// ListObjects 列出对象
//
// 使用 BucketManager.ListFiles 分页遍历，通过 marker 处理分页。
// 当 recursive=false 时使用 "/" 分隔符模拟目录列表。
func (c *ClientImpl) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.list")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.prefix", prefix),
		attribute.Bool("object_store.recursive", recursive),
	)

	bucketManager := storage.NewBucketManager(c.mac, nil)

	var objects []objectx.Object
	marker := ""
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	for {
		entries, _, nextMarker, hasNext, err := bucketManager.ListFiles(bucket, prefix, delimiter, marker, 1000)
		if err != nil {
			return nil, fmt.Errorf("qiniu: list objects %s/%s: %w", bucket, prefix, err)
		}

		for _, entry := range entries {
			objects = append(objects, objectx.Object{
				Key:     entry.Key,
				Size:    entry.Fsize,
				ETag:    entry.Hash,
				LastMod: time.Unix(entry.PutTime/10000000, (entry.PutTime%10000000)*100),
			})
		}

		if !hasNext {
			break
		}
		marker = nextMarker
	}

	return objects, nil
}

// PresignedURL 生成预签名 URL
//
// 对于公开空间使用 MakePublicURL，私有空间使用 MakePrivateURL。
// 私有 URL 的过期时间由 expiry 参数控制（转换为秒数）。
func (c *ClientImpl) PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.presign")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
		attribute.Int64("object_store.expiry_seconds", int64(expiry.Seconds())),
	)

	domain := c.cfg.Endpoint
	if domain == "" {
		return "", fmt.Errorf("qiniu: presigned URL requires a valid Endpoint (download domain)")
	}

	deadline := time.Now().Add(expiry).Unix()
	signedURL := storage.MakePrivateURL(c.mac, domain, key, deadline)
	return signedURL, nil
}

// Exists 检查对象是否存在
//
// 七牛没有原生的 Exists API，通过 Stat 模拟。
// 如果 Stat 返回成功则对象存在，返回 612 错误码（文件不存在）则不存在。
func (c *ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.exists")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	bucketManager := storage.NewBucketManager(c.mac, nil)
	_, err := bucketManager.Stat(bucket, key)
	if err != nil {
		var errInfo *client.ErrorInfo
		if errors.As(err, &errInfo) {
			// 612: 文件不存在, 631: 存储空间不存在, 404: 资源不存在
			if errInfo.Code == 612 || errInfo.Code == 631 || errInfo.Code == 404 {
				return false, nil
			}
		}
		return false, fmt.Errorf("qiniu: check exists %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

// HealthCheck 健康检查
func (c *ClientImpl) HealthCheck() error {
	return nil
}

// Close 关闭客户端（七牛 SDK 无需显式关闭）
func (c *ClientImpl) Close() error {
	return nil
}

// Driver 返回驱动名称
func (c *ClientImpl) Driver() string {
	return "qiniu"
}

func init() {
	objectx.RegisterCreator("qiniu", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
