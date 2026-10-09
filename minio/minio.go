// objectx/minio: MinIO 对象存储驱动
package minio

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	"github.com/gospacex/traceflag"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ClientImpl MinIO 客户端
type ClientImpl struct {
	client *minio.Client
	cfg    *objectx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
	ht     *http.Transport // underlying HTTP transport for CloseIdleConnections
}

// New 创建 MinIO 客户端
func New(cfg *objectx.Config) (*ClientImpl, error) {
	if cfg.Driver != "minio" {
		return nil, fmt.Errorf("minio: expected driver 'minio', got %q", cfg.Driver)
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	ht := &http.Transport{}
	if cfg.TLSEnabled || cfg.UseSSL {
		ht.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	minioOpts := &minio.Options{
		Creds:     credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:    cfg.UseSSL || cfg.TLSEnabled,
		Transport: ht,
	}
	var client *minio.Client
	if err := withRetry(func() error {
		var err2 error
		client, err2 = minio.New(endpoint, minioOpts)
		return err2
	}, 3); err != nil {
		return nil, fmt.Errorf("objectx/minio: create client: %w", err)
	}

	c := &ClientImpl{
		client: client,
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg != nil && cfg.EnableTrace, "github.com/gospacex/objectx/minio", "objectx"),
		logger: logx.GetGlobal(),
		ht:     ht,
	}
	c.logger.InfoContext(context.Background(), "objectx/minio initialized", logx.String("endpoint", endpoint))
	return c, nil
}

func (c *ClientImpl) PutObject(ctx context.Context, bucket, key string, data io.Reader, size int64, opts ...objectx.PutOption) error {
	var span trace.Span
	if c.cfg != nil && c.cfg.EnableTrace {
		ctx, span = c.tracer.Start(ctx, "object_store.put")
		defer span.End()
		span.SetAttributes(
			attribute.String("object_store.bucket", bucket),
			attribute.String("object_store.key", key),
			attribute.Int64("object_store.size", size),
		)
	}

	popts := objectx.PutOptions{}
	for _, opt := range opts {
		opt(&popts)
	}

	_, err := c.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
		ContentType:  popts.ContentType,
		UserMetadata: popts.Metadata,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/minio put failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return err
	}
	c.logger.InfoContext(ctx, "objectx/minio put done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(size)))
	return nil
}

func (c *ClientImpl) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *objectx.Object, error) {
	var span trace.Span
	if c.cfg != nil && c.cfg.EnableTrace {
		ctx, span = c.tracer.Start(ctx, "object_store.get")
		defer span.End()
		span.SetAttributes(
			attribute.String("object_store.bucket", bucket),
			attribute.String("object_store.key", key),
		)
	}

	stat, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, err
	}

	obj := &objectx.Object{
		Key:     key,
		Size:    stat.Size,
		ETag:    stat.ETag,
		LastMod: stat.LastModified,
	}
	reader, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/minio get failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return nil, nil, err
	}
	c.logger.InfoContext(ctx, "objectx/minio get done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(stat.Size)))
	return reader, obj, nil
}

func (c *ClientImpl) DeleteObject(ctx context.Context, bucket, key string) error {
	var span trace.Span
	if c.cfg != nil && c.cfg.EnableTrace {
		ctx, span = c.tracer.Start(ctx, "object_store.delete")
		defer span.End()
		span.SetAttributes(
			attribute.String("object_store.bucket", bucket),
			attribute.String("object_store.key", key),
		)
	}
	if err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		c.logger.ErrorContext(ctx, "objectx/minio delete failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return err
	}
	c.logger.InfoContext(ctx, "objectx/minio delete done", logx.String("bucket", bucket), logx.String("key", key))
	return nil
}

func (c *ClientImpl) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]objectx.Object, error) {
	var objects []objectx.Object
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: recursive}
	for obj := range c.client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			c.logger.ErrorContext(ctx, "objectx/minio list objects error", logx.Err(obj.Err), logx.String("bucket", bucket))
			continue
		}
		objects = append(objects, objectx.Object{
			Key:     obj.Key,
			Size:    obj.Size,
			LastMod: obj.LastModified,
		})
	}
	c.logger.InfoContext(ctx, "objectx/minio list done", logx.String("bucket", bucket), logx.String("prefix", prefix), logx.Int("count", len(objects)))
	return objects, nil
}

func (c *ClientImpl) PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := c.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		var apiErr minio.ErrorResponse
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case "NoSuchBucket", "NoSuchKey":
				return false, nil
			}
		}
		c.logger.ErrorContext(ctx, "objectx/minio exists check failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return false, err
	}
	return true, nil
}

// MakeBucket 创建桶（幂等：桶已存在时返回 nil）。
// 实现 objectx.BucketManager 可选能力，语义同 rustfs 驱动，见接口注释。
func (c *ClientImpl) MakeBucket(ctx context.Context, bucket string) error {
	ctx, span := c.tracer.Start(ctx, "object_store.make_bucket")
	defer span.End()
	span.SetAttributes(attribute.String("object_store.bucket", bucket))

	if bucket == "" {
		return fmt.Errorf("minio: make bucket: empty bucket name")
	}
	exists, err := c.client.BucketExists(ctx, bucket)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("minio: bucket exists check %s: %w", bucket, err)
	}
	if exists {
		return nil
	}
	if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		span.RecordError(err)
		c.logger.ErrorContext(ctx, "objectx/minio make bucket failed", logx.Err(err), logx.String("bucket", bucket))
		return fmt.Errorf("minio: make bucket %s: %w", bucket, err)
	}
	c.logger.InfoContext(ctx, "objectx/minio bucket created", logx.String("bucket", bucket))
	return nil
}

// HealthCheck 真实探活：列举桶。
// 修复前恒返回 nil（假绿），服务宕机时调用方无法感知。详见 rustfs 驱动同名注释。
func (c *ClientImpl) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.client.ListBuckets(ctx); err != nil {
		return fmt.Errorf("minio: health check: %w", err)
	}
	return nil
}

func (c *ClientImpl) Close() error {
	if c.ht != nil {
		c.ht.CloseIdleConnections()
	}
	return nil
}
func (c *ClientImpl) Driver() string { return "minio" }

func init() {
	objectx.RegisterCreator("minio", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}

func withRetry(fn func() error, maxRetries int) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(100<<uint(i-1)) * time.Millisecond)
		}
		if lastErr = fn(); lastErr == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, lastErr)
}
