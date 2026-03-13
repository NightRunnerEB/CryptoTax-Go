package minio

import (
	"errors"
)

var (
	ErrInvalidConfig   = errors.New("invalid minio config")
	ErrInvalidArgument = errors.New("invalid minio argument")
)

func IsInvalidConfig(err error) bool {
	return errors.Is(err, ErrInvalidConfig)
}

func IsInvalidArgument(err error) bool {
	return errors.Is(err, ErrInvalidArgument)
}
