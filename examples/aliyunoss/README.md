# objectx — 阿里云 OSS 示例
#
# 前置准备（参考 https://help.aliyun.com/product/31815.html）：
#   1. 注册阿里云账号，开通 OSS 服务
#   2. 创建 Bucket，记录 Endpoint（控制台 → Bucket 详情 → 外网 Endpoint）
#   3. 获取 AccessKey ID / AccessKey Secret（控制台 → 右上角头像 → AccessKey 管理）
#   4. Endpoint 示例：oss-cn-hangzhou.aliyuncs.com（不含 https:// 前缀）
#
# 运行步骤：
#   cd examples/aliyunoss
#   cp config-example.yaml config.yaml
#   vim config.yaml   ← 填入真实账号信息
#   go run .
