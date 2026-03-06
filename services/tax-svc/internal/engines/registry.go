package engines

import (
	"fmt"

	"github.com/NightRunner/CryptoTax-Go/services/tax-svc/internal/domain"
)

// Registry maps jurisdiction to concrete engine implementation.
type Registry struct {
	engines map[domain.Jurisdiction]Engine
}

func NewRegistry(items ...Engine) (*Registry, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("engines registry: at least one engine is required")
	}

	out := &Registry{
		engines: make(map[domain.Jurisdiction]Engine, len(items)),
	}

	for _, item := range items {
		if item == nil {
			return nil, fmt.Errorf("engines registry: nil engine")
		}
		j := item.Jurisdiction()
		if err := j.Validate(); err != nil {
			return nil, fmt.Errorf("engines registry: invalid engine jurisdiction: %w", err)
		}
		if _, exists := out.engines[j]; exists {
			return nil, fmt.Errorf("engines registry: duplicate engine for jurisdiction %s", j)
		}
		out.engines[j] = item
	}

	return out, nil
}

func (r *Registry) Resolve(j domain.Jurisdiction) (Engine, bool) {
	if r == nil {
		return nil, false
	}
	engine, ok := r.engines[j]
	return engine, ok
}

func (r *Registry) Supports(j domain.Jurisdiction) bool {
	_, ok := r.Resolve(j)
	return ok
}
