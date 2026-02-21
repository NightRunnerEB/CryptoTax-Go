package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIOStorage(cfg config.MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, apperr.StorageUnavailable("minio init failed", err, map[string]string{
			"endpoint": cfg.Endpoint,
		})
	}

	return &MinIOStorage{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

func (s *MinIOStorage) UploadJSON(ctx context.Context, objectKey string, payload any) error {
	if s == nil || s.client == nil {
		return apperr.Internal("storage client is not initialized", nil, nil)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return apperr.StorageBadResponse("marshal payload failed", err, map[string]string{
			"object_key": objectKey,
		})
	}

	reader := bytes.NewReader(body)
	_, err = s.client.PutObject(ctx, s.bucket, objectKey, reader, int64(len(body)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return apperr.StorageUnavailable("upload object failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}

	return nil
}

func (s *MinIOStorage) PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	if s == nil || s.client == nil {
		return "", apperr.Internal("storage client is not initialized", nil, nil)
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}

	u, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, ttl, url.Values{})
	if err != nil {
		return "", apperr.StorageUnavailable("presign failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}

	return u.String(), nil
}

var _ domain.ObjectStorage = (*MinIOStorage)(nil)
