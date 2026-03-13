package minio

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/url"
	"strings"
	"time"

	minioapi "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client *minioapi.Client
	bucket string
	opts   options
}

func New(ctx context.Context, cfg Config, opts ...Option) (*Client, error) {
	cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
	cfg.AccessKey = strings.TrimSpace(cfg.AccessKey)
	cfg.SecretKey = strings.TrimSpace(cfg.SecretKey)
	cfg.Bucket = strings.TrimSpace(cfg.Bucket)

	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("minio.New: %w: endpoint is required", ErrInvalidConfig)
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("minio.New: %w: access_key and secret_key are required", ErrInvalidConfig)
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("minio.New: %w: bucket is required", ErrInvalidConfig)
	}

	settings := defaultOptions()
	for _, opt := range opts {
		opt(&settings)
	}
	if settings.retryMax <= 0 {
		settings.retryMax = 1
	}

	client, err := minioapi.New(cfg.Endpoint, &minioapi.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio.New: %w", err)
	}

	out := &Client{
		client: client,
		bucket: cfg.Bucket,
		opts:   settings,
	}

	exists, err := out.bucketExists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("minio.New: %w: bucket %q does not exist", ErrInvalidConfig, cfg.Bucket)
	}

	return out, nil
}

func (c *Client) PutBytes(ctx context.Context, objectKey string, body []byte, contentType string) error {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return fmt.Errorf("minio.PutBytes: %w: object_key is required", ErrInvalidArgument)
	}

	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err := runWithRetry(ctx, c, "minio.PutBytes", func(opCtx context.Context) (struct{}, error) {
		reader := bytes.NewReader(body)
		_, putErr := c.client.PutObject(opCtx, c.bucket, objectKey, reader, int64(len(body)), minioapi.PutObjectOptions{
			ContentType: contentType,
		})
		return struct{}{}, putErr
	})
	return err
}

func (c *Client) GetBytes(ctx context.Context, objectKey string) ([]byte, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return nil, fmt.Errorf("minio.GetBytes: %w: object_key is required", ErrInvalidArgument)
	}

	return runWithRetry(ctx, c, "minio.GetBytes", func(opCtx context.Context) ([]byte, error) {
		obj, err := c.client.GetObject(opCtx, c.bucket, objectKey, minioapi.GetObjectOptions{})
		if err != nil {
			return nil, err
		}
		defer obj.Close()
		return io.ReadAll(obj)
	})
}

func (c *Client) PresignGet(ctx context.Context, objectKey string, ttl time.Duration) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", fmt.Errorf("minio.PresignGet: %w: object_key is required", ErrInvalidArgument)
	}
	if ttl <= 0 {
		return "", fmt.Errorf("minio.PresignGet: %w: ttl must be > 0", ErrInvalidArgument)
	}

	return runWithRetry(ctx, c, "minio.PresignGet", func(opCtx context.Context) (string, error) {
		u, err := c.client.PresignedGetObject(opCtx, c.bucket, objectKey, ttl, url.Values{})
		if err != nil {
			return "", err
		}
		return u.String(), nil
	})
}

func (c *Client) Bucket() string {
	return c.bucket
}

func (c *Client) bucketExists(ctx context.Context) (bool, error) {
	return runWithRetry(ctx, c, "minio.BucketExists", func(opCtx context.Context) (bool, error) {
		return c.client.BucketExists(opCtx, c.bucket)
	})
}

func runWithRetry[T any](ctx context.Context, c *Client, op string, fn func(context.Context) (T, error)) (T, error) {
	var zero T

	for attempt := 1; attempt <= c.opts.retryMax; attempt++ {
		opCtx, cancel := c.withRequestTimeout(ctx)
		value, err := fn(opCtx)
		cancel()
		if err == nil {
			return value, nil
		}

		if attempt >= c.opts.retryMax || !isRetryable(err) {
			return zero, fmt.Errorf("%s: %w", op, err)
		}
		if !sleepWithContext(ctx, fullJitterBackoff(c.opts.retryBaseDelay, c.opts.retryMaxDelay, attempt)) {
			return zero, fmt.Errorf("%s: %w", op, ctx.Err())
		}
	}

	return zero, fmt.Errorf("%s: %w", op, errors.New("operation failed after retries"))
}

func (c *Client) withRequestTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.opts.requestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.opts.requestTimeout)
}

func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	resp := minioapi.ToErrorResponse(err)
	if resp.Code == "" {
		return false
	}
	if resp.StatusCode >= 500 || resp.StatusCode == 429 || resp.StatusCode == 408 {
		return true
	}
	switch resp.Code {
	case "SlowDown", "ServiceUnavailable", "InternalError", "RequestTimeout", "OperationTimedOut", "RequestTimeTooSkewed", "TemporaryRedirect":
		return true
	default:
		return false
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func fullJitterBackoff(base, max time.Duration, attempt int) time.Duration {
	if base <= 0 {
		return 0
	}
	if max <= 0 || max < base {
		max = base
	}
	if attempt < 1 {
		attempt = 1
	}

	ceiling := base
	for i := 1; i < attempt; i++ {
		if ceiling >= max/2 {
			ceiling = max
			break
		}
		ceiling *= 2
	}
	if ceiling > max {
		ceiling = max
	}

	n, err := crand.Int(crand.Reader, big.NewInt(ceiling.Nanoseconds()+1))
	if err != nil {
		return ceiling / 2
	}
	return time.Duration(n.Int64())
}
