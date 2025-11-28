package server

import (
	"context"
	"hauslet/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-pkgz/lgr"
	"gorm.io/gorm"
)

func setupRoutes(r chi.Router, ctx context.Context, db *gorm.DB, log *lgr.Logger, cfg *config.GlobalConfig) {
	// Route setup code goes here
}
