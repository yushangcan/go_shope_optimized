package main

import (
	"log"

	"go_shope/config"
	"go_shope/dao"
	"go_shope/router"
	"go_shope/service"
)

func main() {
	// 1. 从 config.yaml 和环境变量读取端口、MySQL 连接串、JWT 密钥。
	//    环境变量优先，避免把真实密码提交到 Git。
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	// 2. 建立 MySQL 连接，并让 GORM 自动创建/更新本项目的四张基础表。
	repo, err := dao.New(cfg.MySQL.DSN)
	if err != nil {
		log.Fatal(err)
	}

	// 3. 为每条业务线创建 Service，再交给 router 注册 HTTP 路由。
	//    Service 负责业务规则；router 只负责 HTTP 请求和响应。
	r := router.New(service.NewUserService(repo), service.NewProductService(repo), service.NewActivityService(repo), service.NewOrderService(repo), cfg.JWT.Secret)
	log.Printf("server listening on %s", cfg.Server.Addr)

	// 4. 启动 Gin HTTP 服务器；Run 会一直阻塞，直到程序退出。
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatal(err)
	}
}
