// objectx/tencentcos: 腾讯云 COS 驱动测试
//
// 单元测试（New 校验、工厂注册）始终运行；集成功能套件仅在配置了
// OBJECTX_TEST_TENCENTCOS_* 环境变量时运行（需要真实腾讯云账号）。
package tencentcos

import (
	"testing"

	"github.com/gospacex/objectx"
	"github.com/gospacex/objectx/internal/testutil"
)

// TestNewValidation 构造函数校验（无网络）
func TestNewValidation(t *testing.T) {
	const validEndpoint = "https://test-1250000000.cos.ap-guangzhou.myqcloud.com"
	t.Run("wrong-driver", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "other", Endpoint: validEndpoint, AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(wrong driver) = nil error, want error")
		}
	})
	t.Run("missing-credentials", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "tencentcos", Endpoint: validEndpoint})
		if err == nil {
			t.Fatalf("New(missing creds) = nil error, want error")
		}
	})
	t.Run("missing-endpoint", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "tencentcos", AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(missing endpoint) = nil error, want error")
		}
	})
	t.Run("invalid-endpoint", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "tencentcos", Endpoint: "://bad-url", AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(invalid endpoint) = nil error, want error")
		}
	})
	t.Run("valid", func(t *testing.T) {
		c, err := New(&objectx.Config{Driver: "tencentcos", Endpoint: validEndpoint, AccessKey: "a", SecretKey: "s"})
		if err != nil {
			t.Fatalf("New(valid) error = %v", err)
		}
		defer c.Close()
		if c.Driver() != "tencentcos" {
			t.Fatalf("Driver() = %q, want tencentcos", c.Driver())
		}
	})
}

// TestFactoryRegistration 通过统一工厂创建（无网络）
func TestFactoryRegistration(t *testing.T) {
	c, err := objectx.New(&objectx.Config{Driver: "tencentcos", Endpoint: "https://test-1250000000.cos.ap-guangzhou.myqcloud.com", AccessKey: "a", SecretKey: "s"})
	if err != nil {
		t.Fatalf("objectx.New(tencentcos) error = %v", err)
	}
	defer c.Close()
	if c.Driver() != "tencentcos" {
		t.Fatalf("Driver() = %q, want tencentcos", c.Driver())
	}
}

// TestTencentCOSIntegration 完整功能矩阵（需要真实腾讯云账号）
func TestTencentCOSIntegration(t *testing.T) {
	cfg, ok := testutil.LoadConfigFromEnv(t, "tencentcos")
	if !ok {
		t.Skip("integration skipped: set OBJECTX_TEST_TENCENTCOS_ENDPOINT/AK/SK/BUCKET")
	}
	if cfg.Bucket == "" {
		t.Skip("integration skipped: OBJECTX_TEST_TENCENTCOS_BUCKET required")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	testutil.RunSuite(t, c, cfg.Bucket, testutil.SuiteOptions{VerifyPresignedDownload: false})
}
