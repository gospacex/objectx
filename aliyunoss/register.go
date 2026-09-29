// objectx/aliyunoss - 驱动注册
package aliyunoss

import "github.com/gospacex/objectx"

func init() {
	objectx.RegisterCreator("aliyunoss", func(cfg *objectx.Config) (objectx.BucketClient, error) {
		return New(cfg)
	})
}
