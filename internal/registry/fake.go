package registry

import (
	"context"
	"sync"
)

type MemoryClient struct {
	mu        sync.RWMutex
	instances map[string][]Instance
}

func NewMemoryClient() *MemoryClient {
	return &MemoryClient{instances: make(map[string][]Instance)}
}

func (c *MemoryClient) ListHealthy(_ context.Context, ref ServiceRef) ([]Instance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	raw := c.instances[ref.ServiceName]
	out := make([]Instance, 0, len(raw))
	for _, inst := range raw {
		if ref.Group != "" && inst.Group != ref.Group {
			continue
		}
		if ref.HealthyOnly && (!inst.Healthy || !inst.Enabled || inst.Weight <= 0) {
			continue
		}
		if !matchTags(ref.Tags, inst.Metadata) {
			continue
		}
		out = append(out, inst)
	}
	return out, nil
}

func (c *MemoryClient) Register(_ context.Context, instance Instance) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances[instance.ServiceName] = append(c.instances[instance.ServiceName], instance)
	return nil
}

func (c *MemoryClient) Deregister(_ context.Context, instance Instance) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	raw := c.instances[instance.ServiceName]
	out := raw[:0]
	for _, item := range raw {
		if item.IP == instance.IP && item.Port == instance.Port && item.Group == instance.Group && item.Cluster == instance.Cluster {
			continue
		}
		out = append(out, item)
	}
	c.instances[instance.ServiceName] = out
	return nil
}

func matchTags(expected map[string]string, actual map[string]string) bool {
	for k, v := range expected {
		if actual[k] != v {
			return false
		}
	}
	return true
}
