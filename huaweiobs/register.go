// objectx/huaweiobs - 驱动注册
package huaweiobs

import "github.com/gospacex/objectx"

func init() {
	objectx.RegisterCreator("huaweiobs", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
