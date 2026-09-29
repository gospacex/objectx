// objectx/minio/examples: MinIO 对象存储示例
// 功能：PutObject、GetObject、Exists、HealthCheck
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gospacex/configx"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/objectx"
	_ "github.com/gospacex/objectx/minio"
)

func main() {
	ctx := context.Background()

	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()
	logger.InfoContext(ctx, "objectx minio example started", logx.String("driver", "minio"))

	client, err := objectx.ReadAndCreate(configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.Fatal("create objectx client failed", logx.Err(err))
	}
	defer client.Close()
	logger.InfoContext(ctx, "client driver", logx.String("driver", client.Driver()))

	// HealthCheck
	if err := client.HealthCheck(); err != nil {
		logger.Fatal("health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	// Exists（对象是否存在）
	bucket := "test-bucket"
	exists, err := client.Exists(ctx, bucket, "probe.txt")
	if err != nil {
		logger.Fatal("exists failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "object exists", logx.Any("exists", exists))

	fmt.Println("\n✅ All operations completed successfully.")
}
