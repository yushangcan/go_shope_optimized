package main

import (
	"log"

	"go_shope/config"
	"go_shope/dao"
	"go_shope/router"
	"go_shope/service"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}
	repo, err := dao.New(cfg.MySQL.DSN)
	if err != nil {
		log.Fatal(err)
	}
	r := router.New(service.NewUserService(repo), service.NewProductService(repo), service.NewActivityService(repo), service.NewOrderService(repo), cfg.JWT.Secret)
	log.Printf("server listening on %s", cfg.Server.Addr)
	if err := r.Run(cfg.Server.Addr); err != nil {
		log.Fatal(err)
	}
}
