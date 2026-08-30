package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go_shope/config"
	"go_shope/dao"
	"go_shope/internal/optimized"
	"go_shope/internal/redisstore"
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
	store := redisstore.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.Stream, cfg.Redis.DB)
	w := &optimized.Worker{Service: optimized.New(repo, store), Store: store, Group: "seckill-workers", Name: os.Getenv("WORKER_NAME")}
	if w.Name == "" {
		w.Name = "worker-1"
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
