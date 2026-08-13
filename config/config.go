// Package config 只负责读取配置，不处理任何业务逻辑。
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 的结构要和 config.yaml 的层级保持一致。
type Config struct {
	// Server 下面的 Addr 对应 YAML 中的 server.addr。
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	// MySQL.DSN 是 GORM 连接 MySQL 所需的完整连接字符串。
	MySQL struct {
		DSN string `yaml:"dsn"`
	} `yaml:"mysql"`
	// JWT.Secret 用于签名和验证登录令牌。
	JWT struct {
		Secret string `yaml:"secret"`
	} `yaml:"jwt"`
}

func Load(path string) (Config, error) {
	// cfg 是最终要返回的配置对象；先声明一个空对象再逐步填充。
	var cfg Config
	// 读取 config.yaml 的原始字节内容。
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	// 把 YAML 文本按字段标签解析到 cfg 中。
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	// 如果终端设置了 MYSQL_DSN，就覆盖 YAML 的值。
	// 这是为了让密码只存在当前终端，不写进 Git 文件。
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

// JWT 密钥也优先从环境变量获取。
// 没填写端口时，使用本地开发最常见的 8080。
// 没有数据库连接串，程序无法完成 CRUD，所以立即退出并提示。
// 没有 JWT 密钥就不能安全签发登录令牌，也不允许启动。
