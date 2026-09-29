# objectx — 七牛云 Kodo 示例
#
# 前置准备（参考 https://developer.qiniu.com/kodo）：
#   1. 注册七牛云账号，创建存储空间（Bucket）
#   2. 获取 Access Key / Secret Key（控制台 → 密钥管理）
#   3. 记录 Bucket 名称和上传域名（如 https://up.qiniup.com）
#   4. 记录下载域名（如 https://<bucket>.qiniucdn.com）
#
# 运行步骤：
#   cd examples/qiniu
#   cp config-example.yaml config.yaml
#   vim config.yaml   ← 填入真实账号信息
#   go run .
#
# 注意：
#   - Endpoint 填【上传域名】，七牛上传与下载域名不同，
#     GetObject / PresignedURL 内复用该 Endpoint 作下载域名；
#     若下载走独立 CDN 域名，请按七牛文档另行配置。
#   - 预签名 URL 为私有链接，下载时需携带签名，
#     公开 Bucket 可直接用下载域名访问。
