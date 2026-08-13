package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	MySQL struct {
		DSN string `yaml:"dsn"`
	} `yaml:"mysql"`
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
}

func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	if value := os.Getenv("MYSQL_DSN"); value != "" {
		cfg.MySQL.DSN = value
	}
	if value := os.Getenv("JWT_SECRET"); value != "" {
		cfg.JWT.Secret = value
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
	return cfg, nil
}
