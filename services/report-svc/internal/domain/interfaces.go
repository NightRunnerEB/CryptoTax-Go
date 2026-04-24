package domain

import "context"

type ObjectStorage interface {
	UploadXML(ctx context.Context, objectKey string, xml []byte) error
}
