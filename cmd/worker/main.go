package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"go_shope/config"
	"go_shope/dao"
	"go_shope/internal/mq"
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
	store := redisstore.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	consumer, err := mq.NewConsumer(cfg.MQ.URL, cfg.MQ.Queue, 10)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()
	w := &optimized.Worker{Service: optimized.New(repo, store, nil), Store: store, Consumer: consumer}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := w.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}
