# objectx — 腾讯云 COS 示例
#
# 前置准备（参考 https://cloud.tencent.com/document/product/436）：
#   1. 注册腾讯云账号，开通 COS 服务
#   2. 创建存储桶，记录 Bucket 名称（格式：<name>-<appid>）和 Region
#   3. 获取 SecretID / SecretKey（控制台 → 访问管理 → 访问密钥）
#   4. Endpoint 格式：https://<bucket-name-appid>.cos.<region>.myqcloud.com
#      示例：https://mybucket-1250000000.cos.ap-guangzhou.myqcloud.com
#
# 运行步骤：
#   cd examples/tencentcos
#   cp config-example.yaml config.yaml
#   vim config.yaml   ← 填入真实账号信息
#   go run .
#
# 注意：
#   - Endpoint 需填写 Bucket 完整访问域名（含 appid），而非仅区域端点。
#   - 预签名 URL 的 Bucket 字段只需传桶名（不含 appid），本驱动内部已处理。
