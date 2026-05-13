package discovery

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/internal/registry"
)

const (
	DefaultEndpointMetadataKey = "mcp.endpoint"
	DefaultProtocolMetadataKey = "mcp.protocol"
	DefaultSchemeMetadataKey   = "mcp.scheme"
)

type SelectedInstance struct {
	Instance registry.Instance
	URL      string
	Protocol string
}

type Backend struct {
	discovery registry.Discovery
	cfg       config.DiscoveryConfig
	protocol  string
	lb        *LoadBalancer
	mu        sync.RWMutex
	cached    []registry.Instance
}

func NewBackend(discovery registry.Discovery, cfg config.DiscoveryConfig, lbCfg config.LoadBalanceConfig, expectedProtocol string) *Backend {
	return &Backend{
		discovery: discovery,
		cfg:       applyDiscoveryDefaults(cfg),
		protocol:  expectedProtocol,
		lb:        NewLoadBalancer(lbCfg.Policy),
	}
}

func (b *Backend) Select(ctx context.Context) (SelectedInstance, error) {
	if b == nil || b.discovery == nil {
		return SelectedInstance{}, fmt.Errorf("discovery backend is not configured")
	}

	instances, err := b.discovery.ListHealthy(ctx, registry.ServiceRef{
		ServiceName: b.cfg.ServiceName,
		Group:       b.cfg.Group,
		Clusters:    b.cfg.Clusters,
		HealthyOnly: b.cfg.HealthyOnly,
		Tags:        b.cfg.Tags,
	})
	if err != nil {
		return SelectedInstance{}, err
	}
	instances = filterByProtocol(instances, b.cfg, b.protocol)
	if len(instances) == 0 {
		return SelectedInstance{}, fmt.Errorf("no healthy instance found for service %q", b.cfg.ServiceName)
	}

	b.mu.Lock()
	b.cached = instances
	selected := b.lb.Select(instances)
	b.mu.Unlock()

	if selected.IP == "" || selected.Port == 0 {
		return SelectedInstance{}, fmt.Errorf("selected instance address is empty")
	}
	endpoint := selected.Metadata[b.cfg.EndpointMetadataKey]
	if endpoint == "" {
		return SelectedInstance{}, fmt.Errorf("selected instance missing endpoint metadata %q", b.cfg.EndpointMetadataKey)
	}
	scheme := selected.Metadata[b.cfg.SchemeMetadataKey]
	if scheme == "" {
		scheme = "http"
	}
	protocol := selected.Metadata[b.cfg.ProtocolMetadataKey]
	if protocol == "" {
		protocol = b.protocol
	}
	return SelectedInstance{
		Instance: selected,
		URL:      scheme + "://" + net.JoinHostPort(selected.IP, fmt.Sprintf("%d", selected.Port)) + ensureLeadingSlash(endpoint),
		Protocol: protocol,
	}, nil
}

func (b *Backend) CachedInstances() []registry.Instance {
	if b == nil {
		return nil
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]registry.Instance, len(b.cached))
	copy(out, b.cached)
	return out
}

func applyDiscoveryDefaults(cfg config.DiscoveryConfig) config.DiscoveryConfig {
	if cfg.Registry == "" {
		cfg.Registry = "nacos"
	}
	if cfg.EndpointMetadataKey == "" {
		cfg.EndpointMetadataKey = DefaultEndpointMetadataKey
	}
	if cfg.ProtocolMetadataKey == "" {
		cfg.ProtocolMetadataKey = DefaultProtocolMetadataKey
	}
	if cfg.SchemeMetadataKey == "" {
		cfg.SchemeMetadataKey = DefaultSchemeMetadataKey
	}
	return cfg
}

func filterByProtocol(instances []registry.Instance, cfg config.DiscoveryConfig, expectedProtocol string) []registry.Instance {
	if expectedProtocol == "" {
		return instances
	}
	key := cfg.ProtocolMetadataKey
	if key == "" {
		key = DefaultProtocolMetadataKey
	}
	out := make([]registry.Instance, 0, len(instances))
	for _, instance := range instances {
		if protocol := instance.Metadata[key]; protocol != "" && protocol != expectedProtocol {
			continue
		}
		out = append(out, instance)
	}
	return out
}

func ensureLeadingSlash(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
