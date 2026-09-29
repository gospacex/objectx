// objectx/tencentcos - 驱动注册
package tencentcos

import "github.com/gospacex/objectx"

func init() {
	objectx.RegisterCreator("tencentcos", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
