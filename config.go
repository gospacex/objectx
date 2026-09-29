// objectx/config.go — 统一配置集成（configx + ReadAndCreate）
package objectx

import (
	"fmt"
	"os"
	"time"

	"github.com/gospacex/configx"
)

// StandaloneConfig 单实例对象存储配置（本地/viper，适用于单机部署）
type StandaloneConfig struct {
	Driver          string        `yaml:"driver" json:"driver" mapstructure:"driver"`
	Name            string        `yaml:"name" json:"name" mapstructure:"name"`
	Endpoint        string        `yaml:"endpoint" json:"endpoint" mapstructure:"endpoint"`
	AccessKey       string        `yaml:"access_key" json:"access_key" mapstructure:"access_key"`
	SecretKey       string        `yaml:"secret_key" json:"secret_key" mapstructure:"secret_key"`
	Bucket          string        `yaml:"bucket" json:"bucket" mapstructure:"bucket"`
	UseSSL          bool          `yaml:"use_ssl" json:"use_ssl" mapstructure:"use_ssl"`
	MaxUploadSize   int64         `yaml:"max_upload_size" json:"max_upload_size" mapstructure:"max_upload_size"`
	PresignedExpiry time.Duration `yaml:"presigned_expiry" json:"presigned_expiry" mapstructure:"presigned_expiry"`
	// EnableTrace 统一调用链开关（configx 配置键 enable_trace）
	EnableTrace bool `yaml:"enable_trace" json:"enable_trace" mapstructure:"enable_trace"`
}

// ClusterConfig 集群对象存储配置（Nacos/Apollo 远程配置中心，适用于多节点部署）
type ClusterConfig struct {
	Driver          string        `yaml:"driver" json:"driver" mapstructure:"driver"`
	Name            string        `yaml:"name" json:"name" mapstructure:"name"`
	Endpoint        string        `yaml:"endpoint" json:"endpoint" mapstructure:"endpoint"`
	AccessKey       string        `yaml:"access_key" json:"access_key" mapstructure:"access_key"`
	SecretKey       string        `yaml:"secret_key" json:"secret_key" mapstructure:"secret_key"`
	Bucket          string        `yaml:"bucket" json:"bucket" mapstructure:"bucket"`
	UseSSL          bool          `yaml:"use_ssl" json:"use_ssl" mapstructure:"use_ssl"`
	MaxUploadSize   int64         `yaml:"max_upload_size" json:"max_upload_size" mapstructure:"max_upload_size"`
	PresignedExpiry time.Duration `yaml:"presigned_expiry" json:"presigned_expiry" mapstructure:"presigned_expiry"`
	// Namespace 命名空间（多租户隔离）
	Namespace string `yaml:"namespace" json:"namespace" mapstructure:"namespace"`
	// WatchEnabled 是否监听配置变更
	WatchEnabled bool `yaml:"watch_enabled" json:"watch_enabled" mapstructure:"watch_enabled"`
	// EnableTrace 统一调用链开关（configx 配置键 enable_trace）
	EnableTrace bool `yaml:"enable_trace" json:"enable_trace" mapstructure:"enable_trace"`
}

// ObjectSection 对应 configx.yaml 中的 objectx 配置节
type ObjectSection struct {
	Standalone []*StandaloneConfig `yaml:"standalone"`
	Cluster    []*ClusterConfig    `yaml:"cluster"`
}

// ReadConfig 加载统一配置句柄（所有 x 系列 SDK 共享同一配置源）
func ReadConfig(opts ...configx.ConfigOptions) error {
	_, err := configx.ReadConfig(opts...)
	if err != nil {
		return err
	}
	return nil
}

// ReadAndCreate 从 configx 读取 objectx 配置并创建对象存储客户端
func ReadAndCreate(opts ...configx.ConfigOptions) (BucketClient, error) {
	c, err := configx.ReadConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("objectx: failed to read config: %w", err)
	}

	section := &ObjectSection{}
	if err := c.Unmarshal("objectx", section); err != nil {
		return nil, fmt.Errorf("objectx: failed to unmarshal objectx section: %w", err)
	}

	var cfg *Config
	for _, cc := range section.Cluster {
		cfg = clusterToConfig(cc)
		break
	}
	if cfg == nil {
		for _, sc := range section.Standalone {
			cfg = standaloneToConfig(sc)
			break
		}
	}

	if cfg == nil {
		return nil, fmt.Errorf("objectx: no objectx instance configured")
	}

	return New(cfg)
}

// MustReadAndCreate 类似 ReadAndCreate，失败时 panic
func MustReadAndCreate(opts ...configx.ConfigOptions) BucketClient {
	client, err := ReadAndCreate(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "objectx: failed to initialize object store: %v\n", err)
		os.Exit(1)
	}
	return client
}

func standaloneToConfig(s *StandaloneConfig) *Config {
	return &Config{
		Driver:          s.Driver,
		Name:            s.Name,
		Endpoint:        s.Endpoint,
		AccessKey:       s.AccessKey,
		SecretKey:       s.SecretKey,
		Bucket:          s.Bucket,
		UseSSL:          s.UseSSL,
		MaxUploadSize:   s.MaxUploadSize,
		PresignedExpiry: s.PresignedExpiry,
		EnableTrace:     s.EnableTrace,
	}
}

func clusterToConfig(c *ClusterConfig) *Config {
	return &Config{
		Driver:          c.Driver,
		Name:            c.Name,
		Endpoint:        c.Endpoint,
		AccessKey:       c.AccessKey,
		SecretKey:       c.SecretKey,
		Bucket:          c.Bucket,
		UseSSL:          c.UseSSL,
		MaxUploadSize:   c.MaxUploadSize,
		PresignedExpiry: c.PresignedExpiry,
		EnableTrace:     c.EnableTrace,
	}
}
