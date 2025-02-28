package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hafiztri123/document-api/config"
	"github.com/hafiztri123/document-api/internal/api"
	"github.com/hafiztri123/document-api/internal/database"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)


func main() {
	if err := initConfig(); err != nil {
		log.Fatalf("[ERROR] Initializing config: %v", err)
	}

	logger, err := initLogger()
	if err != nil {
		log.Fatalf("[ERROR] Initializing logger: %v", err)
	}

	//Flush the logger
	defer logger.Sync()

	zap.ReplaceGlobals(logger)

	if viper.GetString(config.ENVIRONMENT) == config.ENV_PROD {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.NewConnection()
	if err != nil {
		logger.Fatal("[ERROR] Failed to connected to dabatase", zap.Error(err))
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", viper.GetString(config.REDIS_HOST), viper.GetInt(config.REDIS_PORT)),
		Password: viper.GetString(config.REDIS_PASSWORD),
		DB: viper.GetInt(config.REDIS_DB),
	})

	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		logger.Fatal("Failed to connect to redis", zap.Error(err))
	}


	router := gin.New()
	router.Use(gin.Recovery())

	api.SetupRoutes(router, db, redisClient, logger)

	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", viper.GetInt(config.SERVER_PORT)),
		Handler: router,
	}

	go func() {
		logger.Info("Starting server", zap.String("address", srv.Addr), zap.String("environment", viper.GetString(config.ENVIRONMENT)))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Error starting server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("[ERROR] Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited properly")
}


func initConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	viper.SetDefault(config.ENVIRONMENT, config.ENV_DEV)
	viper.SetDefault(config.SERVER_PORT, 8080)
	viper.SetDefault(config.DB_MAX_IDLE_CONNECTIONS, 10)
	viper.SetDefault(config.DB_MAX_OPEN_CONNECTIONS, 100)
	viper.SetDefault(config.DB_CONNECTION_MAX_LIFETIME, "1h")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	viper.AutomaticEnv()

	return nil
}

func initLogger() (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if viper.GetString(config.ENVIRONMENT) == config.ENV_PROD {
		logger, err = zap.NewProduction()
	} else {
		logger, err = zap.NewDevelopment()
	}

	return logger, err
}


