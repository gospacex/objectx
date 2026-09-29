// objectx/rustfs: RustFS 对象存储驱动
//
// RustFS 是完全 S3 兼容的本地对象存储。由于其 S3 兼容性，本驱动复用
// minio-go SDK 进行通信，仅将 driver 名称注册为 "rustfs"，与 minio 驱动
// 的区别仅限于注册名称和驱动标识。
package rustfs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	"github.com/gospacex/traceflag"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel/attribute"
)

// ClientImpl RustFS 客户端（复用 minio-go SDK）
type ClientImpl struct {
	client *minio.Client
	cfg    *objectx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

// New 创建 RustFS 客户端
func New(cfg *objectx.Config) (*ClientImpl, error) {
	if cfg.Driver != "rustfs" {
		return nil, fmt.Errorf("rustfs: expected driver 'rustfs', got %q", cfg.Driver)
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "localhost:9000"
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("rustfs: create client: %w", err)
	}

	c := &ClientImpl{
		client: client,
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg != nil && cfg.EnableTrace, "github.com/gospacex/objectx/rustfs", "objectx"),
		logger: logx.GetGlobal(),
	}
	c.logger.InfoContext(context.Background(), "objectx/rustfs initialized", logx.String("endpoint", endpoint))
	return c, nil
}

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

	_, err := c.client.PutObject(ctx, bucket, key, data, size, minio.PutObjectOptions{
		ContentType:  popts.ContentType,
		UserMetadata: popts.Metadata,
	})
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/rustfs put failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return err
	}
	c.logger.InfoContext(ctx, "objectx/rustfs put done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(size)))
	return nil
}

func (c *ClientImpl) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	stat, err := c.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("rustfs: stat object: %w", err)
	}

	obj := &objectx.Object{
		Key:     key,
		Size:    stat.Size,
		ETag:    stat.ETag,
		LastMod: stat.LastModified,
	}
	reader, err := c.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/rustfs get failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return nil, nil, fmt.Errorf("rustfs: get object: %w", err)
	}
	c.logger.InfoContext(ctx, "objectx/rustfs get done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(stat.Size)))
	return reader, obj, nil
}

func (c *ClientImpl) DeleteObject(ctx context.Context, bucket, key string) error {
	ctx, span := c.tracer.Start(ctx, "object_store.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)
	if err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		c.logger.ErrorContext(ctx, "objectx/rustfs delete failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return err
	}
	c.logger.InfoContext(ctx, "objectx/rustfs delete done", logx.String("bucket", bucket), logx.String("key", key))
	return nil
}

func (c *ClientImpl) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]objectx.Object, error) {
	var objects []objectx.Object
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: recursive}
	for obj := range c.client.ListObjects(ctx, bucket, opts) {
		if obj.Err != nil {
			c.logger.ErrorContext(ctx, "objectx/rustfs list objects error", logx.Err(obj.Err), logx.String("bucket", bucket))
			continue
		}
		objects = append(objects, objectx.Object{
			Key:     obj.Key,
			Size:    obj.Size,
			LastMod: obj.LastModified,
		})
	}
	c.logger.InfoContext(ctx, "objectx/rustfs list done", logx.String("bucket", bucket), logx.String("prefix", prefix), logx.Int("count", len(objects)))
	return objects, nil
}

func (c *ClientImpl) PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := c.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("rustfs: presigned url: %w", err)
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
		return false, fmt.Errorf("rustfs: check exists: %w", err)
	}
	return true, nil
}

func (c *ClientImpl) HealthCheck() error { return nil }
func (c *ClientImpl) Close() error       { return nil }
func (c *ClientImpl) Driver() string     { return "rustfs" }

func init() {
	objectx.RegisterCreator("rustfs", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
