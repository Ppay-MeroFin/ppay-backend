package rails

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

type Registry struct {
	mu    sync.RWMutex
	rails map[string]Rail
}

func NewRegistry() *Registry {
	return &Registry{
		rails: make(map[string]Rail),
	}
}

func (r *Registry) Register(rail Rail) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := rail.ID()
	if _, exists := r.rails[id]; exists {
		return fmt.Errorf("rail %s already registered", id)
	}

	r.rails[id] = rail
	log.Printf("registered rail: %s (%s)", rail.DisplayName(), id)
	return nil
}

func (r *Registry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rails[id]; !exists {
		return fmt.Errorf("rail %s not found", id)
	}

	delete(r.rails, id)
	log.Printf("unregistered rail: %s", id)
	return nil
}

func (r *Registry) Get(id string) (Rail, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rail, ok := r.rails[id]
	return rail, ok
}

func (r *Registry) List() []Rail {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Rail, 0, len(r.rails))
	for _, rail := range r.rails {
		out = append(out, rail)
	}
	return out
}

func (r *Registry) HealthSummary(ctx context.Context) []RailStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]RailStatus, 0, len(r.rails))

	for _, rail := range r.rails {
		start := time.Now()
		h, err := rail.HealthCheck(ctx)
		latency := time.Since(start)

		status := RailStatus{
			Name:        rail.ID(),
			Healthy:     err == nil && h != nil && h.Healthy,
			Latency:     latency,
			LastChecked: time.Now(),
			Error:       "",
		}

		if h != nil {
			if h.Latency > 0 {
				status.Latency = h.Latency
			}
			if !h.LastChecked.IsZero() {
				status.LastChecked = h.LastChecked
			}
			status.Healthy = status.Healthy && h.Healthy
			status.Error = h.Details
		}

		if err != nil {
			status.Healthy = false
			status.Error = err.Error()
		}

		out = append(out, status)
	}

	return out
}
