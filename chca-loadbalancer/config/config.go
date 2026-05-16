package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// BackendConfig represents a single backend server entry.
type BackendConfig struct {
	URL    string `mapstructure:"url"`
	Weight int    `mapstructure:"weight"`
}

// RingConfig holds consistent hash ring settings.
type RingConfig struct {
	VirtualNodes int `mapstructure:"virtual_nodes"`
}

// CongestionConfig defines overload thresholds and monitoring intervals.
type CongestionConfig struct {
	MaxConnections     int     `mapstructure:"max_connections"`
	MaxGPUUtilization  float64 `mapstructure:"max_gpu_utilization"`
	MaxCPUUtilization  float64 `mapstructure:"max_cpu_utilization"`
	MonitorIntervalSec int     `mapstructure:"monitor_interval_seconds"`
}

// HealthConfig defines health-checking parameters.
type HealthConfig struct {
	CheckIntervalSec int    `mapstructure:"check_interval_seconds"`
	TimeoutSec       int    `mapstructure:"timeout_seconds"`
	Endpoint         string `mapstructure:"endpoint"`
}

// ServerConfig holds the listener address.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level string `mapstructure:"level"`
}

// Config is the top-level configuration structure.
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Backends   []BackendConfig  `mapstructure:"backends"`
	Ring       RingConfig       `mapstructure:"ring"`
	Congestion CongestionConfig `mapstructure:"congestion"`
	Health     HealthConfig     `mapstructure:"health"`
	Logging    LoggingConfig    `mapstructure:"logging"`
}

// Load reads the configuration from config.yaml using Viper.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	// Defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("ring.virtual_nodes", 150)
	v.SetDefault("congestion.max_connections", 100)
	v.SetDefault("congestion.max_gpu_utilization", 90.0)
	v.SetDefault("congestion.max_cpu_utilization", 80.0)
	v.SetDefault("congestion.monitor_interval_seconds", 5)
	v.SetDefault("health.check_interval_seconds", 5)
	v.SetDefault("health.timeout_seconds", 2)
	v.SetDefault("health.endpoint", "/health")
	v.SetDefault("logging.level", "info")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if len(cfg.Backends) == 0 {
		return nil, fmt.Errorf("at least one backend must be configured")
	}

	return &cfg, nil
}
