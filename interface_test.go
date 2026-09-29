// objectx: 统一接口、工厂与选项单元测试（无网络）
//
// 覆盖 DRIVER_MATRIX.md 中工厂侧的用例：
//   - Config.FillDefaults 默认值
//   - New 的 nil / 未注册 driver 校验与 creator 错误包装
//   - RegisterCreator 工厂往返
//   - WithContentType / WithMetadata 函数选项
package objectx

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// stubBucketClient 实现 BucketClient，用于工厂/选项测试。
type stubBucketClient struct{}

func (stubBucketClient) PutObject(ctx context.Context, bucket, key string, r io.Reader, size int64, opts ...PutOption) error {
	return nil
}
func (stubBucketClient) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, *Object, error) {
	return nil, nil, nil
}
func (stubBucketClient) DeleteObject(ctx context.Context, bucket, key string) error { return nil }
func (stubBucketClient) ListObjects(ctx context.Context, bucket, prefix string, recursive bool) ([]Object, error) {
	return nil, nil
}
func (stubBucketClient) PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	return "", nil
}
func (stubBucketClient) Exists(ctx context.Context, bucket, key string) (bool, error) {
	return false, nil
}
func (stubBucketClient) HealthCheck() error { return nil }
func (stubBucketClient) Close() error       { return nil }
func (stubBucketClient) Driver() string     { return "stub" }

// TestConfigFillDefaults Config 默认值填充
func TestConfigFillDefaults(t *testing.T) {
	tests := []struct {
		name       string
		cfg        Config
		wantExpiry time.Duration
		wantName   string
	}{
		{name: "empty-gets-defaults", cfg: Config{}, wantExpiry: time.Hour, wantName: "default"},
		{name: "negative-expiry-resets", cfg: Config{Name: "prod", PresignedExpiry: -time.Second}, wantExpiry: time.Hour, wantName: "prod"},
		{name: "keeps-set-values", cfg: Config{Name: "prod", PresignedExpiry: 5 * time.Minute}, wantExpiry: 5 * time.Minute, wantName: "prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			cfg.FillDefaults()
			if cfg.PresignedExpiry != tt.wantExpiry {
				t.Fatalf("PresignedExpiry = %v, want %v", cfg.PresignedExpiry, tt.wantExpiry)
			}
			if cfg.Name != tt.wantName {
				t.Fatalf("Name = %q, want %q", cfg.Name, tt.wantName)
			}
		})
	}
}

// TestNewValidation 工厂输入校验
func TestNewValidation(t *testing.T) {
	t.Run("nil-config", func(t *testing.T) {
		_, err := New(nil)
		if err == nil {
			t.Fatalf("New(nil) = nil error, want error")
		}
	})
	t.Run("unregistered-driver", func(t *testing.T) {
		cfg := &Config{Driver: "no-such-driver-registered"}
		_, err := New(cfg)
		if err == nil {
			t.Fatalf("New(unregistered) = nil error, want error")
		}
		if !strings.Contains(err.Error(), cfg.Driver) {
			t.Fatalf("New(unregistered) error %q 未包含驱动名 %q", err, cfg.Driver)
		}
	})
}

// TestNewRegisteredDriver 注册的 creator 可通过统一工厂创建并返回正确 Driver
func TestNewRegisteredDriver(t *testing.T) {
	const driver = "stub-registered"
	RegisterCreator(driver, func(*Config) (BucketClient, error) { return stubBucketClient{}, nil })

	c, err := New(&Config{Driver: driver})
	if err != nil {
		t.Fatalf("New(registered) error = %v", err)
	}
	defer c.Close()
	if c.Driver() != "stub" {
		t.Fatalf("Driver() = %q, want stub", c.Driver())
	}
}

// TestNewPropagatesCreatorError creator 返回的错误应被 %w 包装，可被 errors.Is 识别
func TestNewPropagatesCreatorError(t *testing.T) {
	const driver = "stub-creator-error"
	wantErr := errors.New("creator boom")
	RegisterCreator(driver, func(*Config) (BucketClient, error) { return nil, wantErr })

	_, err := New(&Config{Driver: driver})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want wrap of %v (errors.Is)", err, wantErr)
	}
}

// TestPutOptions 函数选项构造 PutOptions，堆叠时互不污染
func TestPutOptions(t *testing.T) {
	o := &PutOptions{}
	WithContentType("text/plain; charset=utf-8")(o)
	WithMetadata(map[string]string{"owner": "hyx"})(o)

	if o.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("ContentType = %q, want text/plain", o.ContentType)
	}
	if o.Metadata["owner"] != "hyx" {
		t.Fatalf("Metadata = %v, want owner=hyx", o.Metadata)
	}

	// 选项堆叠：单独应用 WithMetadata 不污染 ContentType，反之亦然
	onlyMeta := &PutOptions{}
	WithMetadata(map[string]string{"a": "b"})(onlyMeta)
	if onlyMeta.ContentType != "" {
		t.Fatalf("ContentType = %q after only WithMetadata, want empty", onlyMeta.ContentType)
	}
	onlyCT := &PutOptions{}
	WithContentType("application/json")(onlyCT)
	if onlyCT.Metadata != nil {
		t.Fatalf("Metadata = %v after only WithContentType, want nil", onlyCT.Metadata)
	}
}
