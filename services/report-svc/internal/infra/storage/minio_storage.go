package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func (s *MinIOStorage) DownloadJSON(ctx context.Context, objectKey string, out any) error {
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

	obj, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return apperr.StorageUnavailable("download object failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}
	defer obj.Close()

	body, err := io.ReadAll(obj)
	if err != nil {
		return apperr.StorageUnavailable("read object failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}

	if err := json.Unmarshal(body, out); err != nil {
		return apperr.StorageBadResponse("decode dataset failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}
	return nil
}

func (s *MinIOStorage) UploadPDF(ctx context.Context, objectKey string, pdf []byte) error {
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
	if len(pdf) == 0 {
		return apperr.InvalidArgument("empty pdf payload", nil, apperr.FieldViolation{
			Field:       "pdf",
			Description: "required",
		})
	}

	_, err := s.client.PutObject(ctx, s.bucket, objectKey, bytes.NewReader(pdf), int64(len(pdf)), minio.PutObjectOptions{
		ContentType: "application/pdf",
	})
	if err != nil {
		return apperr.StorageUnavailable("upload pdf failed", err, map[string]string{
			"bucket":     s.bucket,
			"object_key": objectKey,
		})
	}
	return nil
}

var _ domain.ObjectStorage = (*MinIOStorage)(nil)
