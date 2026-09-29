// objectx/tencentcos: 腾讯云 COS 对象存储驱动
//
// Config 字段映射：
//   - Endpoint → Bucket 完整访问域名（如 https://<bucketname-appid>.cos.<region>.myqcloud.com）
//   - AccessKey → SecretID
//   - SecretKey → SecretKey
//   - Bucket    → 存储桶名称（<name>-<appid> 格式），仅用于日志/标识
package tencentcos

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	"github.com/gospacex/traceflag"
	"github.com/tencentyun/cos-go-sdk-v5"
	"go.opentelemetry.io/otel/attribute"
)

// ClientImpl 腾讯云 COS 客户端
type ClientImpl struct {
	client *cos.Client
	cfg    *objectx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

// New 创建腾讯云 COS 客户端
func New(cfg *objectx.Config) (*ClientImpl, error) {
	if cfg.Driver != "tencentcos" {
		return nil, fmt.Errorf("tencentcos: expected driver 'tencentcos', got %q", cfg.Driver)
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("tencentcos: AccessKey (SecretID) and SecretKey are required")
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("tencentcos: Endpoint (bucket URL) is required, e.g. https://<bucket-appid>.cos.<region>.myqcloud.com")
	}

	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("tencentcos: invalid Endpoint %q: %w", cfg.Endpoint, err)
	}

	baseURL := &cos.BaseURL{BucketURL: u}
	transport := &cos.AuthorizationTransport{
		SecretID:  cfg.AccessKey,
		SecretKey: cfg.SecretKey,
	}
	httpClient := &http.Client{Transport: transport}

	c := &ClientImpl{
		client: cos.NewClient(baseURL, httpClient),
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg != nil && cfg.EnableTrace, "github.com/gospacex/objectx/tencentcos", "objectx"),
		logger: logx.GetGlobal(),
	}
	c.logger.InfoContext(context.Background(), "objectx/tencentcos initialized", logx.String("endpoint", cfg.Endpoint))
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

	putOpt := &cos.ObjectPutOptions{}
	putOpt.ObjectPutHeaderOptions = &cos.ObjectPutHeaderOptions{ContentLength: size}
	if popts.ContentType != "" {
		putOpt.ContentType = popts.ContentType
	}
	if len(popts.Metadata) > 0 {
		meta := &http.Header{}
		for k, v := range popts.Metadata {
			meta.Add(k, v)
		}
		putOpt.XCosMetaXXX = meta
	}

	_, err := c.client.Object.Put(ctx, key, data, putOpt)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/tencentcos put failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("tencentcos: put object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/tencentcos put done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(size)))
	return nil
}

func (c *ClientImpl) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	resp, err := c.client.Object.Get(ctx, key, nil)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/tencentcos get failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return nil, nil, fmt.Errorf("tencentcos: get object %s/%s: %w", bucket, key, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("tencentcos: get object %s/%s: unexpected status %d", bucket, key, resp.StatusCode)
	}

	obj := &objectx.Object{
		Key:  key,
		Size: resp.ContentLength,
		ETag: resp.Header.Get("ETag"),
	}
	if lm := resp.Header.Get("Last-Modified"); lm != "" {
		if t, err := time.Parse(time.RFC1123, lm); err == nil {
			obj.LastMod = t
		}
	}
	return resp.Body, obj, nil
}

func (c *ClientImpl) DeleteObject(ctx context.Context, bucket, key string) error {
	ctx, span := c.tracer.Start(ctx, "object_store.delete")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	_, err := c.client.Object.Delete(ctx, key)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/tencentcos delete failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("tencentcos: delete object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/tencentcos delete done", logx.String("bucket", bucket), logx.String("key", key))
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
	marker := ""
	delimiter := ""
	if !recursive {
		delimiter = "/"
	}

	for {
		opt := &cos.BucketGetOptions{
			Prefix:    prefix,
			Delimiter: delimiter,
			Marker:    marker,
			MaxKeys:   1000,
		}
		res, _, err := c.client.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("tencentcos: list objects %s/%s: %w", bucket, prefix, err)
		}

		for _, obj := range res.Contents {
			o := objectx.Object{
				Key:  obj.Key,
				Size: obj.Size,
				ETag: obj.ETag,
			}
			if obj.LastModified != "" {
				if t, err := time.Parse(time.RFC3339, obj.LastModified); err == nil {
					o.LastMod = t
				}
			}
			objects = append(objects, o)
		}

		if !res.IsTruncated {
			break
		}
		marker = res.NextMarker
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

	u, err := c.client.Object.GetPresignedURL(ctx, http.MethodGet, key, c.cfg.AccessKey, c.cfg.SecretKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("tencentcos: presigned url %s/%s: %w", bucket, key, err)
	}
	return u.String(), nil
}

func (c *ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.exists")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	resp, err := c.client.Object.Head(ctx, key, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, fmt.Errorf("tencentcos: check exists %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

func (c *ClientImpl) HealthCheck() error { return nil }
func (c *ClientImpl) Close() error       { return nil }
func (c *ClientImpl) Driver() string     { return "tencentcos" }

func init() {
	objectx.RegisterCreator("tencentcos", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
