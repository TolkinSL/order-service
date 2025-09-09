package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/cache"
	"order-service/internal/config"
	"order-service/internal/handlers"
	"order-service/internal/kafka"
	"order-service/internal/repository"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	cfg := config.LoadConfig()
	logger.Infof("Configuration loaded: DB=%s:%s, Kafka=%v, Server=:%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Kafka.Brokers, cfg.Server.Port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User,
		cfg.Database.Password, cfg.Database.DBName, cfg.Database.SSLMode)

	repo, err := repository.NewPostgresRepository(dsn, logger)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer repo.Close()
	logger.Info("Database connection established")

	memCache := cache.NewMemoryCache(logger)

	logger.Info("Restoring cache from database...")
	if err := memCache.LoadFromRepository(repo); err != nil {
		logger.Errorf("Failed to load cache from repository: %v", err)
	}

	consumer, err := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.GroupID, []string{cfg.Kafka.Topic}, logger)
	if err != nil {
		logger.Fatalf("Failed to create Kafka consumer: %v", err)
	}
	defer consumer.Stop()

	orderHandler := kafka.NewOrderHandler(repo, memCache, logger)
	consumer.AddHandler(orderHandler)

	httpHandler := handlers.NewHTTPHandler(memCache, repo, logger)
	router := httpHandler.SetupRoutes()

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		logger.Info("Starting Kafka consumer...")

		go func() {
			if err := consumer.Start(); err != nil {
				logger.Errorf("Kafka consumer error: %v", err)
				cancel()
			}
		}()

		<-gCtx.Done()
		logger.Info("Stopping Kafka consumer...")
		return consumer.Stop()
	})

	g.Go(func() error {
		logger.Infof("Starting HTTP server on port %s", cfg.Server.Port)

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Errorf("HTTP server error: %v", err)
				cancel()
			}
		}()

		<-gCtx.Done()
		logger.Info("Stopping HTTP server...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		return server.Shutdown(shutdownCtx)
	})

	select {
	case sig := <-sigChan:
		logger.Infof("Received signal: %v", sig)
	case <-gCtx.Done():
		logger.Info("Context cancelled")
	}

	logger.Info("Initiating graceful shutdown...")
	cancel()

	if err := g.Wait(); err != nil {
		logger.Errorf("Service shutdown error: %v", err)
	}

	logger.Info("Service stopped gracefully")
}
