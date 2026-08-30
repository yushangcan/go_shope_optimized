// Package config loads application settings from YAML and environment variables.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Config mirrors the config.yaml hierarchy. mapstructure tags are used by
// Viper when it maps config values into this Go struct.
type Config struct {
	Server struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"server"`
	MySQL struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"mysql"`
	JWT struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"jwt"`
	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
		Stream   string `mapstructure:"stream"`
	} `mapstructure:"redis"`
}

// Load reads YAML first, then lets MYSQL_DSN and JWT_SECRET override YAML.
// This means secrets can stay outside source control in Docker/PowerShell.
func Load(path string) (Config, error) {
	var cfg Config

	// A new instance prevents settings from leaking across tests or reloads.
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}

	// Bind these dotted keys to the existing environment variable names.
	// When an environment variable is set, it takes precedence over config.yaml.
	if err := v.BindEnv("mysql.dsn", "MYSQL_DSN"); err != nil {
		return cfg, fmt.Errorf("bind MYSQL_DSN: %w", err)
	}
	if err := v.BindEnv("jwt.secret", "JWT_SECRET"); err != nil {
		return cfg, fmt.Errorf("bind JWT_SECRET: %w", err)
	}
	if err := v.BindEnv("redis.addr", "REDIS_ADDR"); err != nil {
		return cfg, fmt.Errorf("bind REDIS_ADDR: %w", err)
	}
	if err := v.BindEnv("redis.password", "REDIS_PASSWORD"); err != nil {
		return cfg, fmt.Errorf("bind REDIS_PASSWORD: %w", err)
	}
	if err := v.BindEnv("redis.stream", "REDIS_STREAM"); err != nil {
		return cfg, fmt.Errorf("bind REDIS_STREAM: %w", err)
	}

	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.MySQL.DSN == "" {
		return cfg, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.JWT.Secret == "" {
		return cfg, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.Redis.Addr == "" {
		cfg.Redis.Addr = "127.0.0.1:6379"
	}
	if cfg.Redis.Stream == "" {
		cfg.Redis.Stream = "seckill:stream:orders"
	}
	return cfg, nil
}
