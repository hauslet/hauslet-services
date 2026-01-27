package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type BucketName string

type R2Storage struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucketName    string
}

func NewR2Storage(client *s3.Client, bucketName string) *R2Storage {
	presignClient := s3.NewPresignClient(client)

	return &R2Storage{
		client:        client,
		presignClient: presignClient,
		bucketName:    bucketName,
	}
}

// GenerateSignedUploadURL generates a presigned URL for uploading a file
func (r *R2Storage) GenerateSignedUploadURL(ctx context.Context,
	objectKey string, contentType string, expiresIn time.Duration) (string, error) {
	putObjectParams := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}

	preSignResult, err := r.presignClient.PresignPutObject(ctx, putObjectParams,
		func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned upload URL: %w", err)

	}

	return preSignResult.URL, nil
}

// GenerateSignedDownloadURL generates a presigned URL for downloading a file
func (r *R2Storage) GenerateSignedDownloadURL(ctx context.Context,
	objectKey string, expiresIn time.Duration) (string, error) {
	getObjectParams := &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(objectKey),
	}

	preSignResult, err := r.presignClient.PresignGetObject(ctx, getObjectParams,
		func(opts *s3.PresignOptions) {
			opts.Expires = expiresIn
		})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned download URL: %w", err)
	}

	return preSignResult.URL, nil
}

// DeleteObject deletes an object from the R2 bucket
func (r *R2Storage) DeleteObject(ctx context.Context, objectKey string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		return fmt.Errorf("failed to delete object %s: %w", objectKey, err)
	}
	return nil
}

// ObjectExists checks if an object exists in the bucket
func (r *R2Storage) ObjectExists(ctx context.Context, objectKey string) (bool, error) {
	_, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(objectKey),
	})

	if err != nil {
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if object %s exists: %w", objectKey, err)
	}

	return true, nil
}

// ListObjects returns object keys that match the provided prefix.
func (r *R2Storage) ListObjects(ctx context.Context, prefix string) ([]string, error) {
	if strings.TrimSpace(prefix) == "" {
		return nil, fmt.Errorf("prefix is required")
	}

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucketName),
		Prefix: aws.String(prefix),
	}
	paginator := s3.NewListObjectsV2Paginator(r.client, input)

	var keys []string
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects with prefix %s: %w", prefix, err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || *obj.Key == "" {
				continue
			}
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// UploadObject uploads an object to the bucket
func (r *R2Storage) UploadObject(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload object %s: %w", key, err)
	}
	return nil
}

// GetObject downloads an object from the bucket and returns its bytes and content type.
func (r *R2Storage) GetObject(ctx context.Context, key string) ([]byte, string, error) {
	resp, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read object %s: %w", key, err)
	}

	contentType := ""
	if resp.ContentType != nil {
		contentType = *resp.ContentType
	}

	return data, contentType, nil
}

// GetBucketName returns the name of the r2 bucket
func (c *R2Storage) GetBucketName() string {
	return c.bucketName
}
