// objectx/minio - 驱动注册
package minio

import "github.com/gospacex/objectx"

func init() {
	objectx.RegisterCreator("minio", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
