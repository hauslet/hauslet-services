package setup

import (
	"context"

	"hauslet/config"
	"hauslet/internal/platform/queue"
)

// GetActiveSubjects builds the list of NATS subjects needed based on config.
func GetActiveSubjects(cfg *config.GlobalConfig) []string {
	subjects := []string{cfg.YAML.Queue.Subjects["email"]}

	if s := cfg.YAML.Queue.Subjects["media_thumbnail"]; s != "" {
		subjects = append(subjects, s)
	}
	if s := cfg.YAML.Queue.Subjects["media_cleanup"]; s != "" {
		subjects = append(subjects, s)
	}
	if s := cfg.YAML.Queue.Subjects["ai_moderation"]; s != "" {
		subjects = append(subjects, s)
	}
	return subjects
}

// InitQueue initializes the queue client.
func InitQueue(ctx context.Context, cfg *config.GlobalConfig, subjects []string) (*queue.Client, error) {
	return queue.New(ctx, cfg.Infra.NATS.URL, cfg.YAML.Queue.StreamName, subjects)
}
