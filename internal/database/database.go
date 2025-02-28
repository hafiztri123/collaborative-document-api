package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hafiztri123/document-api/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewConnection() (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(createDataSource()), createGormConfig())
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	setConnectionPool(sqlDB)


	return db, nil
}

func setConnectionPool(sqlDB *sql.DB) {
	sqlDB.SetMaxIdleConns(viper.GetInt(config.DB_MAX_IDLE_CONNECTIONS))
	sqlDB.SetMaxOpenConns(viper.GetInt(config.DB_MAX_OPEN_CONNECTIONS))

	maxLifetime, err := time.ParseDuration(viper.GetString(config.DB_CONNECTION_MAX_LIFETIME))
	if err != nil {
		zap.L().Warn("Invalid connection_max_lifetime, using default 1h", zap.Error(err))
		maxLifetime = time.Hour
	}
	sqlDB.SetConnMaxLifetime(maxLifetime)
}

func createGormConfig() *gorm.Config {

	logLevel := logger.Silent
	if viper.GetString(config.ENVIRONMENT) == config.ENV_DEV {
		logLevel = logger.Info
	}

	return &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

}

func createDataSource() string {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		viper.GetString(config.DB_HOST),
		viper.GetInt(config.DB_PORT),
		viper.GetString(config.DB_USERNAME),
		viper.GetString(config.DB_PASSWORD),
		viper.GetString(config.DB_NAME),
		viper.GetString(config.DB_SSL_MODE),
	)

	return dsn



}
