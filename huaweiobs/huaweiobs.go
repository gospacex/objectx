// objectx/huaweiobs: 华为云 OBS 对象存储驱动
package huaweiobs

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
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"go.opentelemetry.io/otel/attribute"
)

// ClientImpl 华为云 OBS 客户端
type ClientImpl struct {
	client *obs.ObsClient
	cfg    *objectx.Config
	tracer *traceflag.Tracer
	logger logx.Logger
}

// New 创建华为云 OBS 客户端
func New(cfg *objectx.Config) (*ClientImpl, error) {
	if cfg.Driver != "huaweiobs" {
		return nil, fmt.Errorf("huaweiobs: expected driver 'huaweiobs', got %q", cfg.Driver)
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("huaweiobs: endpoint is required")
	}

	obsClient, err := obs.New(cfg.AccessKey, cfg.SecretKey, cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("huaweiobs: create client: %w", err)
	}

	c := &ClientImpl{
		client: obsClient,
		cfg:    cfg,
		tracer: traceflag.NewTracer(cfg != nil && cfg.EnableTrace, "github.com/gospacex/objectx/huaweiobs", "objectx"),
		logger: logx.GetGlobal(),
	}
	c.logger.InfoContext(context.Background(), "objectx/huaweiobs initialized", logx.String("endpoint", cfg.Endpoint))
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

	// go1.22 不支持嵌入字段的复合字面量，需逐字段赋值
	input := &obs.PutObjectInput{}
	input.Bucket = bucket
	input.Key = key
	input.Body = data
	input.ContentLength = size
	if popts.ContentType != "" {
		input.ContentType = popts.ContentType
	}
	if len(popts.Metadata) > 0 {
		input.Metadata = popts.Metadata
	}

	_, err := c.client.PutObject(input)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/huaweiobs put failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("huaweiobs: put object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/huaweiobs put done", logx.String("bucket", bucket), logx.String("key", key), logx.Int("size", int(size)))
	return nil
}

func (c *ClientImpl) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *objectx.Object, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.get")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	input := &obs.GetObjectInput{}
	input.Bucket = bucket
	input.Key = key

	output, err := c.client.GetObject(input)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/huaweiobs get failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return nil, nil, fmt.Errorf("huaweiobs: get object %s/%s: %w", bucket, key, err)
	}

	obj := &objectx.Object{
		Key:     key,
		Size:    output.ContentLength,
		ETag:    output.ETag,
		LastMod: output.LastModified,
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

	input := &obs.DeleteObjectInput{
		Bucket: bucket,
		Key:    key,
	}

	_, err := c.client.DeleteObject(input)
	if err != nil {
		c.logger.ErrorContext(ctx, "objectx/huaweiobs delete failed", logx.Err(err), logx.String("bucket", bucket), logx.String("key", key))
		return fmt.Errorf("huaweiobs: delete object %s/%s: %w", bucket, key, err)
	}
	c.logger.InfoContext(ctx, "objectx/huaweiobs delete done", logx.String("bucket", bucket), logx.String("key", key))
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

	input := &obs.ListObjectsInput{}
	input.Bucket = bucket
	input.Prefix = prefix
	input.MaxKeys = 1000
	if !recursive {
		input.Delimiter = "/"
	}

	for {
		output, err := c.client.ListObjects(input)
		if err != nil {
			return nil, fmt.Errorf("huaweiobs: list objects %s/%s: %w", bucket, prefix, err)
		}

		for _, obj := range output.Contents {
			objects = append(objects, objectx.Object{
				Key:     obj.Key,
				Size:    obj.Size,
				ETag:    obj.ETag,
				LastMod: obj.LastModified,
			})
		}

		if !output.IsTruncated {
			break
		}
		input.Marker = output.NextMarker
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

	input := &obs.CreateSignedUrlInput{
		Method:  obs.HttpMethodGet,
		Bucket:  bucket,
		Key:     key,
		Expires: int(expiry.Seconds()),
	}

	output, err := c.client.CreateSignedUrl(input)
	if err != nil {
		return "", fmt.Errorf("huaweiobs: presigned url %s/%s: %w", bucket, key, err)
	}
	return output.SignedUrl, nil
}

func (c *ClientImpl) Exists(ctx context.Context, bucket, key string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "object_store.exists")
	defer span.End()
	span.SetAttributes(
		attribute.String("object_store.bucket", bucket),
		attribute.String("object_store.key", key),
	)

	input := &obs.HeadObjectInput{
		Bucket: bucket,
		Key:    key,
	}

	_, err := c.client.HeadObject(input)
	if err != nil {
		var obsErr obs.ObsError
		if errors.As(err, &obsErr) {
			// NoSuchKey / NoSuchBucket 等 Code，或 StatusCode 404
			if obsErr.StatusCode == 404 || obsErr.Code == "NoSuchKey" || obsErr.Code == "NoSuchBucket" {
				return false, nil
			}
		}
		return false, fmt.Errorf("huaweiobs: check exists %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

func (c *ClientImpl) HealthCheck() error { return nil }
func (c *ClientImpl) Close() error       { return nil }
func (c *ClientImpl) Driver() string     { return "huaweiobs" }

func init() {
	objectx.RegisterCreator("huaweiobs", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
