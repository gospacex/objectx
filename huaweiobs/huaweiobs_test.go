// objectx/huaweiobs: 华为云 OBS 驱动测试
//
// 单元测试（New 校验、工厂注册）始终运行；集成功能套件仅在配置了
// OBJECTX_TEST_HUAWEOBS_* 环境变量时运行（需要真实华为云账号）。
package huaweiobs

import (
	"testing"

	"github.com/gospacex/objectx"
	"github.com/gospacex/objectx/internal/testutil"
)

// TestNewValidation 构造函数校验（无网络）
func TestNewValidation(t *testing.T) {
	t.Run("wrong-driver", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "other", Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(wrong driver) = nil error, want error")
		}
	})
	t.Run("missing-endpoint", func(t *testing.T) {
		_, err := New(&objectx.Config{Driver: "huaweiobs", AccessKey: "a", SecretKey: "s"})
		if err == nil {
			t.Fatalf("New(missing endpoint) = nil error, want error")
		}
	})
	t.Run("valid", func(t *testing.T) {
		c, err := New(&objectx.Config{Driver: "huaweiobs", Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", AccessKey: "a", SecretKey: "s"})
		if err != nil {
			t.Fatalf("New(valid) error = %v", err)
		}
		defer c.Close()
		if c.Driver() != "huaweiobs" {
			t.Fatalf("Driver() = %q, want huaweiobs", c.Driver())
		}
	})
}

// TestFactoryRegistration 通过统一工厂创建（无网络）
func TestFactoryRegistration(t *testing.T) {
	c, err := objectx.New(&objectx.Config{Driver: "huaweiobs", Endpoint: "https://obs.cn-north-4.myhuaweicloud.com", AccessKey: "a", SecretKey: "s"})
	if err != nil {
		t.Fatalf("objectx.New(huaweiobs) error = %v", err)
	}
	defer c.Close()
	if c.Driver() != "huaweiobs" {
		t.Fatalf("Driver() = %q, want huaweiobs", c.Driver())
	}
}

// TestHuaweiOBSIntegration 完整功能矩阵（需要真实华为云账号）
func TestHuaweiOBSIntegration(t *testing.T) {
	cfg, ok := testutil.LoadConfigFromEnv(t, "huaweiobs")
	if !ok {
		t.Skip("integration skipped: set OBJECTX_TEST_HUAWEOBS_ENDPOINT/AK/SK/BUCKET")
	}
	if cfg.Bucket == "" {
		t.Skip("integration skipped: OBJECTX_TEST_HUAWEOBS_BUCKET required")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer c.Close()

	testutil.RunSuite(t, c, cfg.Bucket, testutil.SuiteOptions{VerifyPresignedDownload: false})
}
