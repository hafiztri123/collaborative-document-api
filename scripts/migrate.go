package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hafiztri123/document-api/config"
	"github.com/spf13/viper"
)

func main() {
	var migrationsPath string
	flag.StringVar(&migrationsPath, "path", "migrations", "Path to migration files")

	var configPath string
	flag.StringVar(&configPath, "config", "config", "Path to config file")

	upCmd := flag.Bool("up", false, "Run migrations up")
	downCmd := flag.Bool("down", false, "Run migrations down")
	versionCmd := flag.Bool("version", false, "Show current migration version")
	flag.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		viper.GetString(config.DB_USERNAME),
		viper.GetString(config.DB_PASSWORD),
		viper.GetString(config.DB_HOST),
		viper.GetInt(config.DB_PORT),
		viper.GetString(config.DB_NAME),
		viper.GetString(config.DB_SSL_MODE),
	)

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		dsn,
	)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	defer m.Close()

	if *upCmd {
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("[ERROR] An error occurred while running migrations: %v", err)
		}
		log.Println("Migrations up completed successfully")
	} else if *downCmd {
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("[ERROR] An error occurred while rolling back migrations: %v", err)
		}
		log.Println("Migrations down completed successfully")
	} else if *versionCmd {
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("[ERROR] An error occurred while getting migration version: %v", err)
		}
		log.Printf("Current version: %d, Dirty: %v\n", version, dirty)
	} else {
		log.Println("No command specified. Use -up, -down, or -version")
		os.Exit(1)
	}
}