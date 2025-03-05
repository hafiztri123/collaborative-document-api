package database

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
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
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
        os.Getenv("PGHOST"),       // Railway provides PGHOST for PostgreSQL host
        GetEnvAsInt("PGPORT", 5432), // Default port is 5432 if not set
        os.Getenv("PGUSER"),       // Railway provides PGUSER for PostgreSQL username
        os.Getenv("PGPASSWORD"),   // Railway provides PGPASSWORD for PostgreSQL password
        os.Getenv("PGDATABASE"),   // Railway provides PGDATABASE for PostgreSQL database name
    )

    return dsn
}



func GetEnvAsInt(key string, defaultValue int) int {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    parsedValue, err := strconv.Atoi(value)
    if err != nil {
        zap.L().Warn(fmt.Sprintf("Invalid value for %s, using default", key), zap.Error(err))
        return defaultValue
    }
    return parsedValue
}