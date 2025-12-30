package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hauslet/internal/platform/redis"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type healthResponse struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func registerHealthRoutes(r chi.Router, db *gorm.DB, rds *redis.RedisClient) {
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeHealth(w, http.StatusOK, "ok", "")
	})

	r.Get("/health/ready", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()

		if err := checkDB(ctx, db); err != nil {
			writeHealth(w, http.StatusServiceUnavailable, "not_ready", "db")
			return
		}
		if err := checkRedis(ctx, rds); err != nil {
			writeHealth(w, http.StatusServiceUnavailable, "not_ready", "redis")
			return
		}

		writeHealth(w, http.StatusOK, "ready", "")
	})
}

func writeHealth(w http.ResponseWriter, status int, message, reason string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status: message,
		Reason: reason,
	})
}

func checkDB(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func checkRedis(ctx context.Context, rds *redis.RedisClient) error {
	if rds == nil || *rds == nil {
		return fmt.Errorf("redis client is nil")
	}
	return (*rds).Ping(ctx).Err()
}
