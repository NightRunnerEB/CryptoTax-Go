package domain

import "context"

type NDFLRenderer interface {
	Render(ctx context.Context, req NDFLRenderRequest) (string, error)
}
