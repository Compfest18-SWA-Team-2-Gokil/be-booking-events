package infrastructure

import (
	"bytes"
	"context"
	"fmt"

	"github.com/ebk-tech/be-booking-events/internal/events/application"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorageProvider struct {
	client    *minio.Client
	bucket    string
	publicURL string // base URL untuk generate link publik, e.g. http://localhost:9000
}

func NewMinIOStorageProvider(endpoint, accessKey, secretKey, bucket, publicURL string, useSSL bool) (*MinIOStorageProvider, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio init: %w", err)
	}
	return &MinIOStorageProvider{client: client, bucket: bucket, publicURL: publicURL}, nil
}

var _ application.StorageProvider = (*MinIOStorageProvider)(nil)

// EnsureBucket membuat bucket jika belum ada dan set policy public-read.
func (s *MinIOStorageProvider) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("cek bucket: %w", err)
	}
	if !exists {
		if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("buat bucket: %w", err)
		}
	}

	// Set public-read policy agar image bisa diakses tanpa auth.
	policy := fmt.Sprintf(`{
		"Version":"2012-10-17",
		"Statement":[{
			"Effect":"Allow",
			"Principal":"*",
			"Action":["s3:GetObject"],
			"Resource":["arn:aws:s3:::%s/*"]
		}]
	}`, s.bucket)

	return s.client.SetBucketPolicy(ctx, s.bucket, policy)
}

func (s *MinIOStorageProvider) UploadImage(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, s.bucket, objectKey,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		return "", fmt.Errorf("upload ke minio: %w", err)
	}

	publicURL := fmt.Sprintf("%s/%s/%s", s.publicURL, s.bucket, objectKey)
	return publicURL, nil
}
