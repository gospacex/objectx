// objectx/aliyunoss: 阿里云 OSS 对象存储驱动 (SDK V2)
//
// Config 字段映射：
//   - Endpoint → OSS 服务 Endpoint（如 https://oss-cn-hangzhou.aliyuncs.com）
//   - AccessKey → AccessKey ID
//   - SecretKey → AccessKey Secret
//   - Bucket    → 存储空间名称（逻辑上不参与拼接域名，由 Endpoint 决定）
package aliyunoss

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	"github.com/gospacex/traceflag"
	"go.opentelemetry.io/otel/attribute"
)

// ClientImpl 阿里云 OSS 客户端
type ClientImpl struct {
	client *oss.Client
	cfg    *objectx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

// New 创建阿里云 OSS 客户端
func New(cfg *objectx.Config) (*ClientImpl, error) {
	if cfg.Driver != "aliyunoss" {
		return nil, fmt.Errorf("aliyunoss: expected driver 'aliyunoss', got %q", cfg.Driver)
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("aliyunoss: Endpoint is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("aliyunoss: AccessKey and SecretKey are required")
	}

	oci := oss.NewConfig()
	oci.Endpoint = oss.Ptr(cfg.Endpoint)
	oci.CredentialsProvider = credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")

	c := &ClientImpl{
		client: oss.NewClient(oci),
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg != nil && cfg.EnableTrace, "github.com/gospacex/objectx/aliyunoss", "objectx"),
		logger: logx.GetGlobal(),
	}
	c.logger.InfoContext(context.Background(), "objectx/aliyunoss initialized", logx.String("endpoint", cfg.Endpoint))
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

	req := &oss.PutObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
		Body:   data,
	}
	if popts.ContentType != "" {
		req.ContentType = oss.Ptr(popts.ContentType)
	}
	if len(popts.Metadata) > 0 {
		req.Metadata = popts.Metadata
	}

	_, err := c.client.PutObject(ctx, req)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/aliyunoss put failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("aliyunoss: put object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/aliyunoss put done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(size)))
	return nil
}

func (c *ClientImpl) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	}
	output, err := c.client.GetObject(ctx, req)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/aliyunoss get failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return nil, nil, fmt.Errorf("aliyunoss: get object %s/%s: %w", bucket, key, err)
	}

	obj := &objectx.Object{
		Key:  key,
		Size: output.ContentLength,
	}
	if output.ETag != nil {
		obj.ETag = *output.ETag
	}
	if output.LastModified != nil {
		obj.LastMod = *output.LastModified
	}
	return output.Body, obj, nil
}

func (c *ClientImpl) DeleteObject(ctx context.Context, bucket, key string) error {
	ctx, span := c.tracer.Start(ctx, "object_store.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	req := &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	}
	_, err := c.client.DeleteObject(ctx, req)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/aliyunoss delete failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("aliyunoss: delete object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/aliyunoss delete done", logx.String("bucket", bucket), logx.String("key", key))
	return nil
}

func (c *ClientImpl) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.list")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.prefix", prefix),
		attribute.Bool("object_store.recursive", recursive),
	)

	var objects []objectx.Object
	continuation := ""
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	for {
		req := &oss.ListObjectsV2Request{
			Bucket:  oss.Ptr(bucket),
			Prefix:  oss.Ptr(prefix),
			MaxKeys: 1000,
		}
		if delimiter != "" {
			req.Delimiter = oss.Ptr(delimiter)
		}
		if continuation != "" {
			req.ContinuationToken = oss.Ptr(continuation)
		}

		output, err := c.client.ListObjectsV2(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("aliyunoss: list objects %s/%s: %w", bucket, prefix, err)
		}

		for _, obj := range output.Contents {
			o := objectx.Object{
				Key:  deref(obj.Key),
				Size: obj.Size,
			}
			if obj.ETag != nil {
				o.ETag = *obj.ETag
			}
			if obj.LastModified != nil {
				o.LastMod = *obj.LastModified
			}
			objects = append(objects, o)
		}

		if !output.IsTruncated {
			break
		}
		if output.NextContinuationToken != nil {
			continuation = *output.NextContinuationToken
		} else {
			break
		}
	}

	return objects, nil
}

func (c *ClientImpl) PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.presign")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
		attribute.Int64("object_store.expiry_seconds", int64(expiry.Seconds())),
	)

	req := &oss.GetObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	}
	result, err := c.client.Presign(ctx, req, oss.PresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("aliyunoss: presigned url %s/%s: %w", bucket, key, err)
	}
	return result.URL, nil
}

func (c *ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.exists")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	req := &oss.HeadObjectRequest{
		Bucket: oss.Ptr(bucket),
		Key:    oss.Ptr(key),
	}
	_, err := c.client.HeadObject(ctx, req)
	if err != nil {
		var serr *oss.ServiceError
		if errors.As(err, &serr) && serr.StatusCode == 404 {
			return false, nil
		}
		return false, fmt.Errorf("aliyunoss: check exists %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

func (c *ClientImpl) HealthCheck() error { return nil }
func (c *ClientImpl) Close() error       { return nil }
func (c *ClientImpl) Driver() string     { return "aliyunoss" }

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func init() {
	objectx.RegisterCreator("aliyunoss", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
