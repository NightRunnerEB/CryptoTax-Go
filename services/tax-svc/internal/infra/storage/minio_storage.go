package storage

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/NightRunner/CryptoTax-Go/pkg/logger"
	miniopkg "github.com/NightRunner/CryptoTax-Go/pkg/minio"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/config"
	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
	apperr "github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain/error"
	"go.uber.org/zap"
)

type MinIOStorage struct {
	client            *miniopkg.Client
	defaultPresignTTL time.Duration
	log               *zap.Logger
}

func NewMinIOStorage(ctx context.Context, cfg config.MinIOConfig) (*MinIOStorage, error) {
	log := logger.FromContext(ctx).With(
		zap.String("component", "minio-storage"),
		zap.String("endpoint", cfg.Endpoint),
		zap.String("bucket", cfg.Bucket),
	)

	client, err := miniopkg.New(
		ctx,
		miniopkg.Config{
			Endpoint:  cfg.Endpoint,
			AccessKey: cfg.AccessKey,
			SecretKey: cfg.SecretKey,
			Bucket:    cfg.Bucket,
			UseSSL:    cfg.UseSSL,
		},
		miniopkg.WithRequestTimeout(cfg.RequestTimeout),
		miniopkg.WithRetry(cfg.RetryMax, cfg.RetryBaseDelay, cfg.RetryMaxDelay),
	)
	if err != nil {
		return nil, mapInitError(err)
	}

	log.Info("minio storage initialized",
		zap.Duration("request_timeout", cfg.RequestTimeout),
		zap.Int("retry_max", cfg.RetryMax),
		zap.Duration("retry_base_delay", cfg.RetryBaseDelay),
		zap.Duration("retry_max_delay", cfg.RetryMaxDelay),
	)

	return &MinIOStorage{
		client:            client,
		defaultPresignTTL: cfg.PresignTTL,
		log:               log,
	}, nil
}

func (s *MinIOStorage) UploadJSON(ctx context.Context, objectKey string, payload any) error {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return apperr.InvalidArgument("invalid object key", nil, apperr.FieldViolation{
			Field:       "object_key",
			Description: "required",
		})
	}
	if payload == nil {
		return apperr.InvalidArgument("invalid payload", nil, apperr.FieldViolation{
			Field:       "payload",
			Description: "required",
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return apperr.StorageBadResponse("marshal payload failed", err, map[string]string{
			"object_key": objectKey,
		})
	}

	s.log.Debug("upload json object",
		zap.String("object_key", objectKey),
		zap.Int("size_bytes", len(body)),
	)

	if err := s.client.PutBytes(ctx, objectKey, body, "application/json"); err != nil {
		return mapRuntimeError("upload json failed", objectKey, err)
	}

	return nil
}

func (s *MinIOStorage) PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", apperr.InvalidArgument("invalid object key", nil, apperr.FieldViolation{
			Field:       "object_key",
			Description: "required",
		})
	}

	url, err := s.client.PresignGet(ctx, objectKey, ttl)
	if err != nil {
		return "", mapRuntimeError("presign failed", objectKey, err)
	}

	return url, nil
}

func mapInitError(err error) error {
	if miniopkg.IsInvalidConfig(err) {
		return apperr.InvalidArgument("invalid minio config", err, apperr.FieldViolation{
			Field:       "minio",
			Description: "invalid configuration",
		})
	}
	return apperr.StorageUnavailable("minio init failed", err, nil)
}

func mapRuntimeError(msg, objectKey string, err error) error {
	if miniopkg.IsInvalidArgument(err) {
		return apperr.InvalidArgument("invalid storage request", err, apperr.FieldViolation{
			Field:       "object_key",
			Description: "invalid",
		})
	}
	return apperr.StorageUnavailable(msg, err, map[string]string{
		"object_key": objectKey,
	})
}

var _ domain.ObjectStorage = (*MinIOStorage)(nil)
