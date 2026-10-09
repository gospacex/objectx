// objectx/examples/minio: MinIO 对象存储示例
// 功能：PutObject、GetObject、DeleteObject、ListObjects、PresignedURL、调用链追踪
package main

import (
	"bytes"
	"context"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"io"
	"log"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	_ "github.com/gospacex/objectx/minio"
)

func initTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint("http://localhost:14268/api/traces")))
	if err != nil {
		return nil, fmt.Errorf("jaeger exporter init failed: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(sdkresource.Default()),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func main() {
	ctx := context.Background()

	// 1. 初始化 logx
	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()

	_, err := initTracer()
	if err != nil {
		logger.InfoContext(ctx, "tracer init skipped", logx.Err(err))
	}
	logger.InfoContext(ctx, "objectx minio example started", logx.String("driver", "minio"))

	// 2. 读取配置并创建客户端
	client, err := objectx.ReadAndCreate(configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.Fatal("create objectx client failed", logx.Err(err))
	}
	defer client.Close()

	// 3. 健康检查
	if err := client.HealthCheck(); err != nil {
		logger.Fatal("health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	bucketName := "test-bucket"
	objectKey := "test-file.txt"

	// 桶必须存在才能 PutObject，BucketClient 没有建桶入口；
	// 支持 objectx.BucketManager 的驱动在这里幂等建桶（对已有桶是 no-op）。
	if bm, ok := client.(objectx.BucketManager); ok {
		if err := bm.MakeBucket(ctx, bucketName); err != nil {
			logger.Fatal("make bucket failed", logx.Err(err))
		}
		logger.InfoContext(ctx, "bucket ready", logx.String("bucket", bucketName))
	} else {
		logger.WarnContext(ctx, "driver has no MakeBucket capability; bucket must pre-exist",
			logx.String("driver", client.Driver()))
	}

	// 4. 上传对象
	logger.InfoContext(ctx, "putting object", logx.String("key", objectKey))
	content := []byte("Hello, MinIO! This is a test file for objectx.")
	if err := client.PutObject(ctx, bucketName, objectKey, bytes.NewReader(content), int64(len(content)), objectx.WithContentType("text/plain")); err != nil {
		logger.Fatal("put object failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object uploaded", logx.Int("size", len(content)))

	// 5. 检查对象是否存在
	logger.InfoContext(ctx, "checking object exists", logx.String("key", objectKey))
	exists, err := client.Exists(ctx, bucketName, objectKey)
	if err != nil {
		logger.Fatal("exists check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object exists", logx.Any("exists", exists))

	// 6. 下载对象
	logger.InfoContext(ctx, "getting object", logx.String("key", objectKey))
	reader, obj, err := client.GetObject(ctx, bucketName, objectKey)
	if err != nil {
		logger.Fatal("get object failed", logx.Err(err))
	}
	defer reader.Close()
	buf, _ := io.ReadAll(reader)
	logger.InfoContext(ctx, "object retrieved", logx.String("content", string(buf)), logx.Int("size", int(obj.Size)))

	// 7. 列出对象
	logger.InfoContext(ctx, "listing objects")
	objects, err := client.ListObjects(ctx, bucketName, "", false)
	if err != nil {
		logger.Fatal("list objects failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "listed objects", logx.Int("count", len(objects)))

	// 8. 生成预签名 URL
	logger.InfoContext(ctx, "generating presigned URL")
	presignedURL, err := client.PresignedURL(ctx, bucketName, objectKey, 24*time.Hour)
	if err != nil {
		logger.Fatal("presigned URL failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "presigned URL generated", logx.String("url", presignedURL))

	// 9. 删除对象
	logger.InfoContext(ctx, "deleting object", logx.String("key", objectKey))
	if err := client.DeleteObject(ctx, bucketName, objectKey); err != nil {
		logger.Fatal("delete object failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object deleted")

	// 10. 验证删除
	exists, _ = client.Exists(ctx, bucketName, objectKey)
	logger.InfoContext(ctx, "object removed", logx.Any("exists", exists))

	fmt.Println("\n✅ All operations completed successfully.")
}
