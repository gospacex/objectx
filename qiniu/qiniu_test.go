// objectx/qiniu: 七牛云 Kodo 驱动测试
//
// 单元测试（New 校验、工厂注册）始终运行；集成功能套件仅在配置了
// OBJECTX_TEST_QINIU_* 环境变量时运行（需要真实七牛账号）。
package qiniu

import (
	"testing"

	"github.com/gospacex/objectx"
	"github.com/gospacex/objectx/internal/testutil"
)

// TestNewValidation 构造函数校验（无网络）
func TestNewValidation(t *testing.T) {
	t.Run("wrong-driver", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "other", AccessKey: "a", SecretKey: "s", Bucket: "b"})
		if err == nil {
			t.Fatalf("New(wrong driver) = nil error, want error")
		}
	})
	t.Run("missing-credentials", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "qiniu", Bucket: "b"})
		if err == nil {
			t.Fatalf("New(missing creds) = nil error, want error")
		}
	})
	t.Run("missing-bucket", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "qiniu", AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(missing bucket) = nil error, want error")
		}
	})
	t.Run("valid", func(t *testing.T) {
		c, err := New(&objectx.Config{Driver: "qiniu", AccessKey: "a", SecretKey: "s", Bucket: "b"})
		if err != nil {
			t.Fatalf("New(valid) error = %v", err)
		}
		defer c.Close()
		if c.Driver() != "qiniu" {
			t.Fatalf("Driver() = %q, want qiniu", c.Driver())
		}
	})
}

// TestFactoryRegistration 通过统一工厂创建（无网络）
func TestFactoryRegistration(t *testing.T) {
	c, err := objectx.New(&objectx.Config{Driver: "qiniu", AccessKey: "a", SecretKey: "s", Bucket: "b"})
	if err != nil {
		t.Fatalf("objectx.New(qiniu) error = %v", err)
	}
	defer c.Close()
	if c.Driver() != "qiniu" {
		t.Fatalf("Driver() = %q, want qiniu", c.Driver())
	}
}

// TestQiniuIntegration 完整功能矩阵（需要真实七牛账号 + 下载域名）
func TestQiniuIntegration(t *testing.T) {
	cfg, ok := testutil.LoadConfigFromEnv(t, "qiniu")
	if !ok {
		t.Skip("integration skipped: set OBJECTX_TEST_QINIU_ENDPOINT/AK/SK/BUCKET")
	}
	if cfg.Bucket == "" {
		t.Skip("integration skipped: OBJECTX_TEST_QINIU_BUCKET required")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	// 公有云不实际下载预签名 URL（依赖 CDN/公开访问配置，可能 403）
	testutil.RunSuite(t, c, cfg.Bucket, testutil.SuiteOptions{VerifyPresignedDownload: false})
}
