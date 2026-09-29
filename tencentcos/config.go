// objectx/tencentcos - 腾讯云 COS 对象存储驱动配置（Standalone + Cluster）
package tencentcos

import (
	"time"

	"github.com/gospacex/objectx"
)

// StandaloneConfig 腾讯云 COS 单实例配置
type StandaloneConfig struct {
	Driver          string        `yaml:"driver" json:"driver"`
	Name            string        `yaml:"name" json:"name"`
	Endpoint        string        `yaml:"endpoint" json:"endpoint"`
	AccessKey       string        `yaml:"access_key" json:"access_key"`
	SecretKey       string        `yaml:"secret_key" json:"secret_key"`
	Bucket          string        `yaml:"bucket" json:"bucket"`
	Region          string        `yaml:"region" json:"region"`
	UseSSL          bool          `yaml:"use_ssl" json:"use_ssl"`
	MaxUploadSize   int64         `yaml:"max_upload_size" json:"max_upload_size"`
	PresignedExpiry time.Duration `yaml:"presigned_expiry" json:"presigned_expiry"`
}

// ClusterConfig 腾讯云 COS 集群配置（多地域容灾）
type ClusterConfig struct {
	Driver          string        `yaml:"driver" json:"driver"`
	Name            string        `yaml:"name" json:"name"`
	Endpoints       []string      `yaml:"endpoints" json:"endpoints"`
	AccessKey       string        `yaml:"access_key" json:"access_key"`
	SecretKey       string        `yaml:"secret_key" json:"secret_key"`
	Bucket          string        `yaml:"bucket" json:"bucket"`
	Region          string        `yaml:"region" json:"region"`
	UseSSL          bool          `yaml:"use_ssl" json:"use_ssl"`
	MaxUploadSize   int64         `yaml:"max_upload_size" json:"max_upload_size"`
	PresignedExpiry time.Duration `yaml:"presigned_expiry" json:"presigned_expiry"`
	// Cluster
	EnableTrace bool `yaml:"enable_trace" json:"enable_trace"`
}

// ToStandaloneConfig 转换为 objectx.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *objectx.Config {
	n := name
	if s.Name != "" {
		n = s.Name
	}
	return &objectx.Config{
		Driver:          "tencentcos",
		Name:            n,
		Endpoint:        s.Endpoint,
		AccessKey:       s.AccessKey,
		SecretKey:       s.SecretKey,
		Bucket:          s.Bucket,
		UseSSL:          s.UseSSL,
		MaxUploadSize:   s.MaxUploadSize,
		PresignedExpiry: s.PresignedExpiry,
	}
}

// ToClusterConfig 转换为 objectx.Config
func (c *ClusterConfig) ToClusterConfig(name string) *objectx.Config {
	n := name
	if c.Name != "" {
		n = c.Name
	}
	endpoint := ""
	if len(c.Endpoints) > 0 {
		endpoint = c.Endpoints[0]
	}
	return &objectx.Config{
		Driver:          "tencentcos",
		Name:            n,
		Endpoint:        endpoint,
		AccessKey:       c.AccessKey,
		SecretKey:       c.SecretKey,
		Bucket:          c.Bucket,
		UseSSL:          c.UseSSL,
		MaxUploadSize:   c.MaxUploadSize,
		PresignedExpiry: c.PresignedExpiry,
	}
}
