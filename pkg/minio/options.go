package minio

import "time"

const (
	defaultRequestTimeout = 10 * time.Second
	defaultRetryMax       = 3
	defaultRetryBase      = 200 * time.Millisecond
	defaultRetryMaxDelay  = 2 * time.Second
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type options struct {
	requestTimeout time.Duration
	retryMax       int
	retryBaseDelay time.Duration
	retryMaxDelay  time.Duration
}

func defaultOptions() options {
	return options{
		requestTimeout: defaultRequestTimeout,
		retryMax:       defaultRetryMax,
		retryBaseDelay: defaultRetryBase,
		retryMaxDelay:  defaultRetryMaxDelay,
	}
}

type Option func(*options)

func WithRequestTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.requestTimeout = timeout
		}
	}
}

func WithRetry(max int, baseDelay, maxDelay time.Duration) Option {
	return func(o *options) {
		if max > 0 {
			o.retryMax = max
		}
		if baseDelay > 0 {
			o.retryBaseDelay = baseDelay
		}
		if maxDelay > 0 {
			o.retryMaxDelay = maxDelay
		}
		if o.retryMaxDelay < o.retryBaseDelay {
			o.retryMaxDelay = o.retryBaseDelay
		}
	}
}
