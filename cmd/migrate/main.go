package main

import (
	"context"
	"log"
	"os"

	"hauslet/cmd/api/setup"
	"hauslet/config"
	"hauslet/internal/platform/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	logger := setup.SetupLogger(env)

	ctx, cancel := context.WithTimeout(context.Background(), setup.InitTimeout)
	defer cancel()

	dbConfig := config.DBConfig{
		DBHost:     mustEnv("DB_HOST"),
		DBUser:     mustEnv("DB_USER"),
		DbName:     mustEnv("DB_NAME"),
		DBPassword: mustEnv("DB_PASSWORD"),
		DBPort:     mustEnv("DB_PORT"),
		DBSslmode:  envOrDefault("DB_SSLMODE", "disable"),
	}

	db, err := database.NewPostgresWithContext(ctx, &dbConfig, env)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	if err := setup.RunMigrations(db, logger); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("ENV %s is required", key)
	}
	return value
}

func envOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
