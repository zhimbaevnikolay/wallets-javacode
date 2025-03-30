package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wallets/internal/config"
	"wallets/internal/http-server/handlers/wallets/addqueue"
	"wallets/internal/http-server/handlers/wallets/create"
	"wallets/internal/http-server/handlers/wallets/getbalance"
	"wallets/internal/lib/prettylog"
	"wallets/internal/lib/sl"
	"wallets/internal/storage"
	"wallets/internal/storage/postgres"
	redis_client "wallets/internal/storage/redisclient"
	redisworker "wallets/internal/storage/rediswoker"

	"github.com/gin-gonic/gin"
)

const (
	envLocal = "local"
	envProd  = "prod"
)

func main() {

	cfg := config.MustLoad()

	log := initLogger(cfg.Env)

	log.Info("starting wallets service")

	log.Debug("debug messages are enabled")

	postgres, err := postgres.New(cfg.Storage)
	if err != nil {
		log.Error("storage initialization failed", sl.Err(err))
		os.Exit(1)
	}

	redisClient, err := redis_client.New(cfg.Redis)
	if err != nil {
		log.Error("redis initilization failed", sl.Err(err))
		os.Exit(1)
	}

	redisWorker := redisworker.New("update:wallet", redisClient.Client)

	storage := storage.NewStorage(postgres, redisClient, redisWorker)

	ctx := context.Background()
	router := gin.New()

	api := router.Group("/api/v1")
	{
		wallet := api.Group("/wallet")
		{
			wallet.POST("", addqueue.New(ctx, log, storage))
			wallet.POST("/create", create.New(ctx, log, storage.DB))

		}

		wallets := api.Group("/wallets")
		{
			wallets.GET("/:uuid", getbalance.New(ctx, log, storage))
		}

	}

	log.Info("starting server...", slog.String("address", cfg.HTTPServer.Address))

	srv := &http.Server{
		Addr:         cfg.HTTPServer.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.Idle_timeout,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("failed to start server", sl.Err(err))
		}
	}()

	log.Info("server started")

	go storage.StartWorker(ctx, log)

	log.Info("worker started")

	<-done
	log.Info("server is shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", sl.Err(err))
	}

	log.Info("server stopped")

}

func initLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			prettylog.NewHandler((&slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
			))

	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		)

	}

	return log
}
