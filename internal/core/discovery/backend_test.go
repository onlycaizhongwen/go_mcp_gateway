package discovery

import (
	"context"
	"testing"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/internal/registry"
)

func TestBackendSelectBuildsURLFromRegistryMetadata(t *testing.T) {
	client := registry.NewMemoryClient()
	err := client.Register(context.Background(), registry.Instance{
		ServiceName: "mock-user-sse",
		Group:       "DEFAULT_GROUP",
		Cluster:     "DEFAULT",
		IP:          "127.0.0.1",
		Port:        5237,
		Weight:      1,
		Healthy:     true,
		Enabled:     true,
		Metadata: map[string]string{
			DefaultEndpointMetadataKey: "/sse",
			DefaultProtocolMetadataKey: "sse",
			DefaultSchemeMetadataKey:   "http",
		},
	})
	if err != nil {
		t.Fatalf("register instance: %v", err)
	}

	backend := NewBackend(client, config.DiscoveryConfig{
		ServiceName: "mock-user-sse",
		Group:       "DEFAULT_GROUP",
		HealthyOnly: true,
	}, config.LoadBalanceConfig{Policy: PolicyRoundRobin}, "sse")

	selected, err := backend.Select(context.Background())
	if err != nil {
		t.Fatalf("select instance: %v", err)
	}
	if selected.URL != "http://127.0.0.1:5237/sse" {
		t.Fatalf("unexpected URL: %s", selected.URL)
	}
	if selected.Protocol != "sse" {
		t.Fatalf("unexpected protocol: %s", selected.Protocol)
	}
}
