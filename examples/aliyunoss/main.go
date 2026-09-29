// objectx/examples/aliyunoss: 阿里云 OSS 对象存储示例
//
// 前置准备（参考 https://help.aliyun.com/product/31815.html）：
//  1. 开通 OSS 服务，创建 Bucket，记录 Endpoint（如 oss-cn-hangzhou.aliyuncs.com）
//  2. 获取 AccessKey ID/AccessKey Secret（控制台 → 访问控制 → 访问密钥）
//  3. 复制 config-example.yaml → config.yaml 并填入真实值
//  4. go run .
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	_ "github.com/gospacex/objectx/aliyunoss"
)

func main() {
	ctx := context.Background()

	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()

	client, err := objectx.ReadAndCreate(configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.Fatal("create objectx client failed", logx.Err(err))
	}
	defer client.Close()

	if err := client.HealthCheck(); err != nil {
		logger.Fatal("health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	bucketName := "test-bucket"
	objectKey := "test-file.txt"

	logger.InfoContext(ctx, "putting object", logx.String("key", objectKey))
	content := []byte("Hello, Aliyun OSS! This is a test file for objectx.")
	if err := client.PutObject(ctx, bucketName, objectKey, bytes.NewReader(content), int64(len(content)), objectx.WithContentType("text/plain")); err != nil {
		logger.Fatal("put object failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object uploaded", logx.Int("size", len(content)))

	logger.InfoContext(ctx, "checking object exists", logx.String("key", objectKey))
	exists, err := client.Exists(ctx, bucketName, objectKey)
	if err != nil {
		logger.Fatal("exists check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object exists", logx.Any("exists", exists))

	logger.InfoContext(ctx, "getting object", logx.String("key", objectKey))
	reader, obj, err := client.GetObject(ctx, bucketName, objectKey)
	if err != nil {
		logger.Fatal("get object failed", logx.Err(err))
	}
	defer reader.Close()
	buf, _ := io.ReadAll(reader)
	logger.InfoContext(ctx, "object retrieved", logx.String("content", string(buf)), logx.Int("size", int(obj.Size)))

	logger.InfoContext(ctx, "listing objects")
	objects, err := client.ListObjects(ctx, bucketName, "", false)
	if err != nil {
		logger.Fatal("list objects failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "listed objects", logx.Int("count", len(objects)))

	logger.InfoContext(ctx, "generating presigned URL")
	presignedURL, err := client.PresignedURL(ctx, bucketName, objectKey, 24*time.Hour)
	if err != nil {
		logger.Fatal("presigned URL failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "presigned URL generated", logx.String("url", presignedURL))

	logger.InfoContext(ctx, "deleting object", logx.String("key", objectKey))
	if err := client.DeleteObject(ctx, bucketName, objectKey); err != nil {
		logger.Fatal("delete object failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object deleted")

	exists, _ = client.Exists(ctx, bucketName, objectKey)
	logger.InfoContext(ctx, "object removed", logx.Any("exists", exists))

	fmt.Println("\n✅ All operations completed successfully.")
}
