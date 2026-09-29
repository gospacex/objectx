// objectx/internal/testutil - 对象存储驱动通用功能测试套件
//
// 所有驱动（minio/rustfs/qiniu/huaweiobs/tencentcos/aliyunoss）共享
// 同一套功能用例，覆盖 DRIVER_MATRIX.md 中的用例矩阵：
//   - 基础 CRUD
//   - 元数据/ContentType
//   - Exists / ListObjects / PresignedURL
//   - 错误处理与健壮性（并发、大对象）
package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gospacex/objectx"
)

// SuiteOptions 功能套件选项
type SuiteOptions struct {
	// VerifyPresignedDownload 是否实际下载预签名 URL 验证可用性。
	// 本地部署驱动（minio/rustfs）设置 true；公有云按环境可选。
	VerifyPresignedDownload bool
	// LargeSize 大对象用例的字节数（默认 1MB）
	LargeSize int64
}

// RunSuite 对给定客户端运行完整功能用例矩阵。
// bucket 需已由调用方确保存在（minio/rustfs 自动创建见各自测试）。
func RunSuite(t *testing.T, c objectx.BucketClient, bucket string, opts SuiteOptions) {
	if opts.LargeSize <= 0 {
		opts.LargeSize = 1024 * 1024 // 1MB
	}
	t.Helper()

	t.Run("driver-name", func(t *testing.T) {
		if got := c.Driver(); got == "" {
			t.Fatalf("Driver() returned empty")
		}
	})
	t.Run("health-check", func(t *testing.T) {
		if err := c.HealthCheck(); err != nil {
			t.Fatalf("HealthCheck() error = %v", err)
		}
	})

	t.Run("crud", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/crud-%d.txt", time.Now().UnixNano())
		payload := []byte("hello objectx crud\n第二行中文内容\n")

		// Put
		if err := c.PutObject(ctx, bucket, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
			t.Fatalf("PutObject() error = %v", err)
		}
		// Get 验证内容
		rc, obj, err := c.GetObject(ctx, bucket, key)
		if err != nil {
			t.Fatalf("GetObject() error = %v", err)
		}
		defer rc.Close()
		got, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read body error = %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("roundtrip mismatch: got %q want %q", got, payload)
		}
		if obj.Key != key {
			t.Fatalf("Object.Key = %q, want %q", obj.Key, key)
		}
		// Delete
		if err := c.DeleteObject(ctx, bucket, key); err != nil {
			t.Fatalf("DeleteObject() error = %v", err)
		}
	})

	t.Run("put-with-metadata", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/meta-%d.txt", time.Now().UnixNano())
		payload := []byte("meta payload")
		meta := map[string]string{"owner": "hyx", "source": "suite"}
		if err := c.PutObject(ctx, bucket, key, bytes.NewReader(payload), int64(len(payload)),
			objectx.WithMetadata(meta), objectx.WithContentType("text/plain; charset=utf-8")); err != nil {
			t.Fatalf("PutObject(meta) error = %v", err)
		}
		defer c.DeleteObject(ctx, bucket, key)
	})

	t.Run("exists-true-false", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/exists-%d.txt", time.Now().UnixNano())
		payload := []byte("exists check")

		// 未上传 → false
		ok, err := c.Exists(ctx, bucket, key)
		if err != nil {
			t.Fatalf("Exists(before) error = %v", err)
		}
		if ok {
			t.Fatalf("Exists(before) = true, want false")
		}

		if err := c.PutObject(ctx, bucket, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
			t.Fatalf("PutObject() error = %v", err)
		}
		// 上传后 → true
		ok, err = c.Exists(ctx, bucket, key)
		if err != nil {
			t.Fatalf("Exists(after) error = %v", err)
		}
		if !ok {
			t.Fatalf("Exists(after) = false, want true")
		}

		if err := c.DeleteObject(ctx, bucket, key); err != nil {
			t.Fatalf("DeleteObject() error = %v", err)
		}
		// 删除后 → false
		ok, err = c.Exists(ctx, bucket, key)
		if err != nil {
			t.Fatalf("Exists(deleted) error = %v", err)
		}
		if ok {
			t.Fatalf("Exists(deleted) = true, want false")
		}
	})

	t.Run("get-nonexistent-errors", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/missing-%d.txt", time.Now().UnixNano())
		_, _, err := c.GetObject(ctx, bucket, key)
		if err == nil {
			t.Fatalf("GetObject(missing) = nil error, want error")
		}
	})

	t.Run("delete-nonexistent", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/delete-missing-%d.txt", time.Now().UnixNano())
		// 各家行为不一（幂等删除 or 报错），只要求不 panic 且返回值可接受
		_ = c.DeleteObject(ctx, bucket, key)
	})

	t.Run("list", func(t *testing.T) {
		ctx := context.Background()
		base := fmt.Sprintf("suite/list-%d", time.Now().UnixNano())
		keys := []string{
			base + "/dir1/a.txt",
			base + "/dir1/b.txt",
			base + "/dir2/c.txt",
			base + "/root.txt",
		}
		payload := []byte("x")
		for _, k := range keys {
			if err := c.PutObject(ctx, bucket, k, bytes.NewReader(payload), int64(len(payload))); err != nil {
				t.Fatalf("PutObject(%s) error = %v", k, err)
			}
		}
		defer func() {
			for _, k := range keys {
				c.DeleteObject(ctx, bucket, k)
			}
		}()

		// list-empty：不存在的前缀
		empty := "suite/__nope__"
		items, err := c.ListObjects(ctx, bucket, empty, true)
		if err != nil {
			t.Fatalf("ListObjects(empty) error = %v", err)
		}
		if len(items) != 0 {
			t.Fatalf("ListObjects(empty) = %d items, want 0", len(items))
		}

		// list-recursive：列出全部 4 个
		items, err = c.ListObjects(ctx, bucket, base, true)
		if err != nil {
			t.Fatalf("ListObjects(recursive) error = %v", err)
		}
		got := map[string]bool{}
		for _, it := range items {
			got[it.Key] = true
		}
		for _, k := range keys {
			if !got[k] {
				t.Fatalf("ListObjects(recursive) missing %q (got %d items)", k, len(items))
			}
		}

		// list-prefix 过滤
		items, err = c.ListObjects(ctx, bucket, base+"/dir2", true)
		if err != nil {
			t.Fatalf("ListObjects(prefix) error = %v", err)
		}
		if len(items) != 1 || items[0].Key != base+"/dir2/c.txt" {
			t.Fatalf("ListObjects(prefix) = %+v, want only dir2/c.txt", items)
		}
	})

	t.Run("presigned-url", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/presign-%d.txt", time.Now().UnixNano())
		payload := []byte("presign payload")
		if err := c.PutObject(ctx, bucket, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
			t.Fatalf("PutObject() error = %v", err)
		}
		defer c.DeleteObject(ctx, bucket, key)

		u, err := c.PresignedURL(ctx, bucket, key, 15*time.Minute)
		if err != nil {
			t.Fatalf("PresignedURL() error = %v", err)
		}
		if u == "" {
			t.Fatalf("PresignedURL() returned empty")
		}

		if opts.VerifyPresignedDownload {
			httpClient := &http.Client{Timeout: 30 * time.Second}
			resp, err := httpClient.Get(u)
			if err != nil {
				t.Fatalf("download presigned URL error = %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("download presigned URL status = %d, want 200", resp.StatusCode)
			}
			got, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("read presigned body error = %v", err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("presigned download mismatch: got %q want %q", got, payload)
			}
		}
	})

	t.Run("large-object", func(t *testing.T) {
		ctx := context.Background()
		key := fmt.Sprintf("suite/large-%d.bin", time.Now().UnixNano())
		payload := bytes.Repeat([]byte("0123456789abcdef"), int(opts.LargeSize/16)+1)
		payload = payload[:opts.LargeSize]
		// 用随机模式避免所有块相同导致的稀疏文件
		for i := 0; i < len(payload); i += 4096 {
			payload[i] = byte(i * 31)
		}

		if err := c.PutObject(ctx, bucket, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
			t.Fatalf("PutObject(large) error = %v", err)
		}
		defer c.DeleteObject(ctx, bucket, key)

		ok, err := c.Exists(ctx, bucket, key)
		if err != nil {
			t.Fatalf("Exists(large) error = %v", err)
		}
		if !ok {
			t.Fatalf("Exists(large) = false, want true")
		}

		rc, obj, err := c.GetObject(ctx, bucket, key)
		if err != nil {
			t.Fatalf("GetObject(large) error = %v", err)
		}
		defer rc.Close()
		got, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read large body error = %v", err)
		}
		if len(got) != len(payload) {
			t.Fatalf("large size mismatch: got %d want %d", len(got), len(payload))
		}
		if obj.Size != int64(len(payload)) {
			t.Fatalf("Object.Size = %d, want %d", obj.Size, len(payload))
		}
	})

	t.Run("concurrent-ops", func(t *testing.T) {
		ctx := context.Background()
		const n = 8
		var wg sync.WaitGroup
		errCh := make(chan error, n*2)
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("suite/concurrent-%d-%d.txt", time.Now().UnixNano()%1000000, i)
				payload := []byte(strings.Repeat(fmt.Sprintf("item-%d;", i), 100))
				if err := c.PutObject(ctx, bucket, key, bytes.NewReader(payload), int64(len(payload))); err != nil {
					errCh <- fmt.Errorf("put %s: %w", key, err)
					return
				}
				rc, _, err := c.GetObject(ctx, bucket, key)
				if err != nil {
					errCh <- fmt.Errorf("get %s: %w", key, err)
					return
				}
				rc.Close()
				if err := c.DeleteObject(ctx, bucket, key); err != nil {
					errCh <- fmt.Errorf("delete %s: %w", key, err)
				}
			}(i)
		}
		wg.Wait()
		close(errCh)
		for err := range errCh {
			if err != nil {
				t.Fatalf("concurrent-op failed: %v", err)
			}
		}
	})
}

// LoadConfigFromEnv 从环境变量构造驱动配置，未配置时返回 nil（调用方跳过集成测试）。
//
// 约定：<DRIVER> 用大写。变量：
//
//	OBJECTX_TEST_<DRIVER>_ENDPOINT  必填
//	OBJECTX_TEST_<DRIVER>_AK        选填
//	OBJECTX_TEST_<DRIVER>_SK        选填
//	OBJECTX_TEST_<DRIVER>_BUCKET    选填
//	OBJECTX_TEST_<DRIVER>_SSL        选填，1 表示 UseSSL
func LoadConfigFromEnv(t *testing.T, driver string) (*objectx.Config, bool) {
	t.Helper()
	upper := strings.ToUpper(driver)
	base := "OBJECTX_TEST_" + upper
	endpoint := envOr(base+"_ENDPOINT", "")
	if endpoint == "" {
		return nil, false
	}
	cfg := &objectx.Config{
		Driver:    driver,
		Endpoint:  endpoint,
		AccessKey: envOr(base+"_AK", ""),
		SecretKey: envOr(base+"_SK", ""),
		Bucket:    envOr(base+"_BUCKET", ""),
		UseSSL:    envOr(base+"_SSL", "") == "1",
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
