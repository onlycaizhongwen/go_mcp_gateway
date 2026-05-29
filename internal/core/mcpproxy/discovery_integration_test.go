package mcpproxy

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/amoylab/unla/internal/common/cnst"
	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/internal/registry"
	nacosregistry "github.com/amoylab/unla/internal/registry/nacos"
	"github.com/amoylab/unla/pkg/mcp"
)

func TestDiscoveryTransportWithLiveNacos(t *testing.T) {
	if os.Getenv("UNLA_LIVE_NACOS_TEST") != "1" {
		t.Skip("set UNLA_LIVE_NACOS_TEST=1 to run live Nacos discovery test")
	}
	serviceName := "mock-user-sse-sdk-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))

	client, err := nacosregistry.NewClient(config.NacosRegistryConfig{
		NamespaceID: "",
		Group:       "DEFAULT_GROUP",
		Clusters:    []string{"DEFAULT"},
		TimeoutMS:   5000,
		CacheDir:    "./data/nacos/cache-test",
		LogDir:      "./logs/nacos-test",
		Servers: []config.NacosServerConfig{
			{IP: "127.0.0.1", Port: 8848, Scheme: "http"},
		},
	})
	if err != nil {
		t.Fatalf("create nacos client: %v", err)
	}
	instance := registry.Instance{
		ServiceName: serviceName,
		Group:       "DEFAULT_GROUP",
		Cluster:     "DEFAULT",
		IP:          "127.0.0.1",
		Port:        5237,
		Weight:      1,
		Healthy:     true,
		Enabled:     true,
		Metadata: map[string]string{
			"mcp.endpoint": "/sse",
			"mcp.host":     "localhost",
			"mcp.protocol": "sse",
			"mcp.scheme":   "http",
		},
	}
	if err := client.Register(context.Background(), instance); err != nil {
		t.Fatalf("register nacos instance: %v", err)
	}
	defer func() { _ = client.Deregister(context.Background(), instance) }()

	transport, err := NewDiscoveryTransport(client, config.MCPServerConfig{
		Type:   string(TypeSSE),
		Name:   "mock-user-sse-sdk",
		Policy: cnst.PolicyOnDemand,
		Discovery: config.DiscoveryConfig{
			Enabled:     true,
			Registry:    "nacos",
			ServiceName: serviceName,
			Group:       "DEFAULT_GROUP",
			HealthyOnly: true,
		},
		LoadBalance: config.LoadBalanceConfig{Policy: "round_robin"},
	})
	if err != nil {
		t.Fatalf("create discovery transport: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tools []mcp.ToolSchema
	for i := 0; i < 10; i++ {
		tools, err = transport.FetchTools(ctx)
		if err == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("fetch tools through discovery: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected tools from discovered MCP server")
	}

	args, _ := json.Marshal(map[string]any{})
	result, err := transport.CallTool(ctx, mcp.CallToolParams{
		Name:      "getTinyImage",
		Arguments: args,
	}, nil)
	if err != nil {
		t.Fatalf("call tool through discovery: %v", err)
	}
	if result == nil || result.IsError || len(result.Content) == 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestDiscoveryTransportWithLiveNacosAutoRegisteredService(t *testing.T) {
	if os.Getenv("UNLA_LIVE_NACOS_AUTOREG_TEST") != "1" {
		t.Skip("set UNLA_LIVE_NACOS_AUTOREG_TEST=1 to run against an auto-registered MCP server")
	}
	serviceName := firstNonEmpty(os.Getenv("UNLA_AUTOREG_SERVICE_NAME"), "mock-user-sse-autoreg-15337")

	client, err := nacosregistry.NewClient(config.NacosRegistryConfig{
		NamespaceID: "",
		Group:       "DEFAULT_GROUP",
		Clusters:    []string{"DEFAULT"},
		TimeoutMS:   5000,
		CacheDir:    "./data/nacos/cache-autoreg-test",
		LogDir:      "./logs/nacos-autoreg-test",
		Servers: []config.NacosServerConfig{
			{IP: "127.0.0.1", Port: 8848, Scheme: "http"},
		},
	})
	if err != nil {
		t.Fatalf("create nacos client: %v", err)
	}

	transport, err := NewDiscoveryTransport(client, config.MCPServerConfig{
		Type:   string(TypeSSE),
		Name:   "mock-user-sse-autoreg",
		Policy: cnst.PolicyOnDemand,
		Discovery: config.DiscoveryConfig{
			Enabled:     true,
			Registry:    "nacos",
			ServiceName: serviceName,
			Group:       "DEFAULT_GROUP",
			HealthyOnly: true,
		},
		LoadBalance: config.LoadBalanceConfig{Policy: "round_robin"},
	})
	if err != nil {
		t.Fatalf("create discovery transport: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tools, err := transport.FetchTools(ctx)
	if err != nil {
		t.Fatalf("fetch tools through auto-registered discovery: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected tools from auto-registered MCP server")
	}

	args, _ := json.Marshal(map[string]any{})
	result, err := transport.CallTool(ctx, mcp.CallToolParams{
		Name:      "getTinyImage",
		Arguments: args,
	}, nil)
	if err != nil {
		t.Fatalf("call tool through auto-registered discovery: %v", err)
	}
	if result == nil || result.IsError || len(result.Content) == 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
