// objectx/rustfs: RustFS 驱动测试
//
// 单元测试（New 校验、工厂注册）始终运行；集成功能套件仅在本地 docker
// RustFS 可用时运行（docker compose up -d，宿主映射 9010:9000）。
package rustfs

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
		c, err := New(&objectx.Config{Driver: "rustfs", AccessKey: "rustfsadmin", SecretKey: "rustfsadmin"})
		if err != nil {
			t.Fatalf("New(valid) error = %v", err)
		}
		defer c.Close()
		if c.Driver() != "rustfs" {
			t.Fatalf("Driver() = %q, want rustfs", c.Driver())
		}
	})
}

// TestFactoryRegistration 通过统一工厂创建（无网络）
func TestFactoryRegistration(t *testing.T) {
	c, err := objectx.New(&objectx.Config{Driver: "rustfs", AccessKey: "a", SecretKey: "s"})
	if err != nil {
		t.Fatalf("objectx.New(rustfs) error = %v", err)
	}
	defer c.Close()
	if c.Driver() != "rustfs" {
		t.Fatalf("Driver() = %q, want rustfs", c.Driver())
	}
}

// TestRustFSIntegration 完整功能矩阵（需要本地 docker RustFS）
//
// 启用方式：
//   - OBJECTX_TEST_DOCKER=1            使用默认 localhost:9010 / rustfsadmin
//   - OBJECTX_TEST_RUSTFS_ENDPOINT=... 自定义 endpoint（覆盖）
func TestRustFSIntegration(t *testing.T) {
	cfg, ok := integrationConfig()
	if !ok {
		t.Skip("integration skipped: set OBJECTX_TEST_DOCKER=1 or OBJECTX_TEST_RUSTFS_ENDPOINT")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	ctx := context.Background()
	bucket := cfg.Bucket
	if bucket == "" {
		bucket = "objectx-test-rustfs"
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
	endpoint := os.Getenv("OBJECTX_TEST_RUSTFS_ENDPOINT")
	if endpoint == "" && os.Getenv("OBJECTX_TEST_DOCKER") == "1" {
		endpoint = "localhost:9010"
	}
	if endpoint == "" {
		return nil, false
	}
	cfg := &objectx.Config{
		Driver:    "rustfs",
		Endpoint:  endpoint,
		AccessKey: envOr("OBJECTX_TEST_RUSTFS_AK", "rustfsadmin"),
		SecretKey: envOr("OBJECTX_TEST_RUSTFS_SK", "rustfsadmin"),
		Bucket:    os.Getenv("OBJECTX_TEST_RUSTFS_BUCKET"),
		UseSSL:    os.Getenv("OBJECTX_TEST_RUSTFS_SSL") == "1",
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
