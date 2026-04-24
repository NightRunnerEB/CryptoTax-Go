package storage

import (
	"bytes"
	"context"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/report-svc/internal/domain/error"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(cfg config.MinIOConfig) (*MinIOStorage, error) {
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, apperr.InvalidArgument("invalid minio bucket", nil, apperr.FieldViolation{
			Field:       "minio.bucket",
			Description: "required",
		})
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, apperr.StorageUnavailable("minio init failed", err, map[string]string{
			"endpoint": cfg.Endpoint,
		})
	}

	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, apperr.StorageUnavailable("check minio bucket failed", err, map[string]string{
			"endpoint": cfg.Endpoint,
			"bucket":   bucket,
		})
	}
	if !exists {
		return nil, apperr.StorageUnavailable("minio bucket does not exist", nil, map[string]string{
			"endpoint": cfg.Endpoint,
			"bucket":   bucket,
		})
	}

	return &MinIOStorage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *MinIOStorage) UploadXML(ctx context.Context, objectKey string, payload []byte) error {
	if s == nil || s.client == nil {
		return apperr.Internal("storage client is not initialized", nil, nil)
	}
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return apperr.InvalidArgument("invalid object key", nil, apperr.FieldViolation{
			Field:       "object_key",
			Description: "required",
		})
	}
	if len(payload) == 0 {
		return apperr.InvalidArgument("empty xml payload", nil, apperr.FieldViolation{
			Field:       "xml",
			Description: "required",
		})
	}

	_, err := s.client.PutObject(ctx, s.bucket, objectKey, bytes.NewReader(payload), int64(len(payload)), minio.PutObjectOptions{
		ContentType: "application/xml",
	})
	if err != nil {
		return apperr.StorageUnavailable("upload xml failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}
	return nil
}

var _ domain.ObjectStorage = (*MinIOStorage)(nil)
