// objectx/qiniu - 驱动注册
package qiniu

import "github.com/gospacex/objectx"

func init() {
	objectx.RegisterCreator("qiniu", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
