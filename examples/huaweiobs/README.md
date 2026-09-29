# objectx — 华为云 OBS 示例
#
# 前置准备（参考 https://support.huaweicloud.com/obs）：
#   1. 注册华为云账号，开通 OBS 服务
#   2. 创建 OBS 桶（Bucket），记录 Region（如 cn-north-4）
#   3. 获取 AK/SK（控制台 → 我的凭证 → 访问密钥）
#   4. Endpoint 格式：obs.<region>.myhuaweicloud.com
#      示例：obs.cn-north-4.myhuaweicloud.com
#
# 运行步骤：
#   cd examples/huaweiobs
#   cp config-example.yaml config.yaml
#   vim config.yaml   ← 填入真实账号信息
#   go run .
