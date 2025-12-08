package storage

import (
	"context"
	"fmt"
	cfg "hauslet/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewR2S3Client creates a fully configured S3 client for Cloudflare R2 storage.
// Returns an S3-compatible client configured with R2 credentials and endpoint.
func NewR2S3Client(cfg cfg.R2Config) (*s3.Client, error) {
	// Load default configuration with R2 credentials
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.AccessKeySecret,
				"",
			),
		),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load Cloudflare R2 config: %w", err)
	}

	// Create S3 client with R2-specific options
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	return client, nil
}
