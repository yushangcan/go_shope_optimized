package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go_shope/config"
	"go_shope/dao"
	"go_shope/internal/mq"
	"go_shope/internal/observability"
	"go_shope/internal/optimized"
	"go_shope/internal/redisstore"
	"go_shope/middleware"
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
	store := redisstore.New(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	publisher, err := mq.New(cfg.MQ.URL, cfg.MQ.Queue)
	if err != nil {
		log.Fatal(err)
	}
	defer publisher.Close()
	optimizedService := optimized.New(repo, store, publisher)
	// Warm Redis from MySQL on startup so a single-machine Redis restart does
	// not leave existing activities unavailable to the admission script.
	if activities, listErr := repo.ListActivities(); listErr == nil {
		for i := range activities {
			if err := optimizedService.PublishActivity(context.Background(), &activities[i]); err != nil {
				log.Printf("publish activity %d to redis: %v", activities[i].ID, err)
			}
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := router.New(service.NewUserService(repo), service.NewProductService(repo), service.NewActivityService(repo), service.NewOrderService(repo), cfg.JWT.Secret)
	r.Use(observability.Middleware())
	r.GET("/livez", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) {
		if err := store.Ping(c); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	r.GET("/metrics", gin.WrapH(observability.Handler()))
	api := r.Group("/api")
	api.Use(middleware.Auth(cfg.JWT.Secret))
	api.POST("/seckill/activities/:id/requests", optimized.NewHandler(optimizedService).Admit)
	api.GET("/seckill/requests/:request_id", optimized.NewHandler(optimizedService).RequestStatus)
	admin := api.Group("/admin")
	admin.Use(middleware.RequireAdmin())
	admin.POST("/seckill/activities/:id/publish", optimized.NewHandler(optimizedService).PublishActivity)

	server := &http.Server{Addr: cfg.Server.Addr, Handler: r, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
