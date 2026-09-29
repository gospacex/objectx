// objectx/minio: MinIO 驱动测试
//
// 单元测试（New 校验、工厂注册）始终运行；集成功能套件仅在本地 docker
// MinIO 可用时运行（docker compose up -d，见仓库根 docker-compose.yml）。
package minio

import (
	"context"
	"os"
	"testing"

	"github.com/gospacex/objectx"
	"github.com/gospacex/objectx/internal/testutil"
	"github.com/minio/minio-go/v7"
)

// TestNewValidation 构造函数校验（无网络）
func TestNewValidation(t *testing.T) {
	t.Run("wrong-driver", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "other", AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(wrong driver) = nil error, want error")
		}
	})
	t.Run("valid-default-endpoint", func(t *testing.T) {
		c, err := New(&objectx.Config{Driver: "minio", AccessKey: "minioadmin", SecretKey: "minioadmin123"})
		if err != nil {
			t.Fatalf("New(valid) error = %v", err)
		}
		defer c.Close()
		if c.Driver() != "minio" {
			t.Fatalf("Driver() = %q, want minio", c.Driver())
		}
	})
}

// TestFactoryRegistration 通过统一工厂创建（无网络）
func TestFactoryRegistration(t *testing.T) {
	c, err := objectx.New(&objectx.Config{Driver: "minio", AccessKey: "a", SecretKey: "s"})
	if err != nil {
		t.Fatalf("objectx.New(minio) error = %v", err)
	}
	defer c.Close()
	if c.Driver() != "minio" {
		t.Fatalf("Driver() = %q, want minio", c.Driver())
	}
}

// TestMinioIntegration 完整功能矩阵（需要本地 docker MinIO）
//
// 启用方式：
//   - OBJECTX_TEST_DOCKER=1            使用默认 localhost:9000 / minioadmin
//   - OBJECTX_TEST_MINIO_ENDPOINT=...  自定义 endpoint（覆盖）
func TestMinioIntegration(t *testing.T) {
	cfg, ok := integrationConfig()
	if !ok {
		t.Skip("integration skipped: set OBJECTX_TEST_DOCKER=1 or OBJECTX_TEST_MINIO_ENDPOINT")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "objectx-test-minio"
	}
	if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		exists, e := c.client.BucketExists(ctx, bucket)
		if e != nil || !exists {
			t.Fatalf("MakeBucket(%s) error = %v", bucket, err)
		}
	}

	testutil.RunSuite(t, c, bucket, testutil.SuiteOptions{VerifyPresignedDownload: true})
}

func integrationConfig() (*objectx.Config, bool) {
	endpoint := os.Getenv("OBJECTX_TEST_MINIO_ENDPOINT")
	if endpoint == "" && os.Getenv("OBJECTX_TEST_DOCKER") == "1" {
		endpoint = "localhost:9000"
	}
	if endpoint == "" {
		return nil, false
	}
	cfg := &objectx.Config{
		Driver:    "minio",
		Endpoint:  endpoint,
		AccessKey: envOr("OBJECTX_TEST_MINIO_AK", "minioadmin"),
		SecretKey: envOr("OBJECTX_TEST_MINIO_SK", "minioadmin123"),
		Bucket:    os.Getenv("OBJECTX_TEST_MINIO_BUCKET"),
		UseSSL:    os.Getenv("OBJECTX_TEST_MINIO_SSL") == "1",
	}
	cfg.FillDefaults()
	return cfg, true
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
