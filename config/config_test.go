package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsYAML(t *testing.T) {
	path := writeTestConfig(t)
	t.Setenv("MYSQL_DSN", "")
	t.Setenv("JWT_SECRET", "")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Server.Addr != ":9090" {
		t.Errorf("Server.Addr = %q, want :9090", cfg.Server.Addr)
	}
	if cfg.MySQL.DSN != "yaml-dsn" {
		t.Errorf("MySQL.DSN = %q, want yaml-dsn", cfg.MySQL.DSN)
	}
	if cfg.JWT.Secret != "yaml-secret" {
		t.Errorf("JWT.Secret = %q, want yaml-secret", cfg.JWT.Secret)
	}
}

func TestLoadEnvironmentOverridesYAML(t *testing.T) {
	path := writeTestConfig(t)
	t.Setenv("MYSQL_DSN", "environment-dsn")
	t.Setenv("JWT_SECRET", "environment-secret")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.MySQL.DSN != "environment-dsn" {
		t.Errorf("MySQL.DSN = %q, want environment-dsn", cfg.MySQL.DSN)
	}
	if cfg.JWT.Secret != "environment-secret" {
		t.Errorf("JWT.Secret = %q, want environment-secret", cfg.JWT.Secret)
	}
}

func writeTestConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "server:\n  addr: ':9090'\nmysql:\n  dsn: yaml-dsn\njwt:\n  secret: yaml-secret\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
