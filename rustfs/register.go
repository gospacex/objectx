// objectx/rustfs - 驱动注册
package rustfs

import "github.com/gospacex/objectx"

func init() {
	objectx.RegisterCreator("rustfs", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
