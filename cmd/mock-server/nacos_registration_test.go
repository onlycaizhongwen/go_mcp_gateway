package main

import "testing"

func TestBuildNacosInstanceDefaultsPortFromSSEAddr(t *testing.T) {
	instance, err := buildNacosInstance(nacosRegistrationOptions{
		ServiceName: "mock-user-sse",
		RegisterIP:  "127.0.0.1",
		MCPHost:     "localhost",
	}, ":5237")
	if err != nil {
		t.Fatalf("build instance: %v", err)
	}
	if instance.ServiceName != "mock-user-sse" {
		t.Fatalf("unexpected service name: %s", instance.ServiceName)
	}
	if instance.Group != "DEFAULT_GROUP" {
		t.Fatalf("unexpected group: %s", instance.Group)
	}
	if instance.Cluster != "DEFAULT" {
		t.Fatalf("unexpected cluster: %s", instance.Cluster)
	}
	if instance.IP != "127.0.0.1" || instance.Port != 5237 {
		t.Fatalf("unexpected address: %s", instance.Address())
	}
	if instance.Metadata["mcp.endpoint"] != "/sse" {
		t.Fatalf("unexpected endpoint metadata: %v", instance.Metadata)
	}
	if instance.Metadata["mcp.protocol"] != "sse" {
		t.Fatalf("unexpected protocol metadata: %v", instance.Metadata)
	}
	if instance.Metadata["mcp.scheme"] != "http" {
		t.Fatalf("unexpected scheme metadata: %v", instance.Metadata)
	}
	if instance.Metadata["mcp.host"] != "localhost" {
		t.Fatalf("unexpected host metadata: %v", instance.Metadata)
	}
}

func TestBuildNacosInstanceOverrides(t *testing.T) {
	instance, err := buildNacosInstance(nacosRegistrationOptions{
		NacosGroup:   "MCP_GROUP",
		NacosCluster: "blue",
		ServiceName:  "user-service",
		RegisterIP:   "10.0.0.7",
		RegisterPort: 9000,
		MCPScheme:    "https",
		MCPProtocol:  "streamable-http",
		MCPEndpoint:  "/mcp",
	}, ":5237")
	if err != nil {
		t.Fatalf("build instance: %v", err)
	}
	if instance.Group != "MCP_GROUP" || instance.Cluster != "blue" {
		t.Fatalf("unexpected group/cluster: %s/%s", instance.Group, instance.Cluster)
	}
	if instance.IP != "10.0.0.7" || instance.Port != 9000 {
		t.Fatalf("unexpected address: %s", instance.Address())
	}
	if _, ok := instance.Metadata["mcp.host"]; ok {
		t.Fatalf("mcp.host should be omitted when not configured: %v", instance.Metadata)
	}
	if instance.Metadata["mcp.protocol"] != "streamable-http" || instance.Metadata["mcp.endpoint"] != "/mcp" {
		t.Fatalf("unexpected metadata: %v", instance.Metadata)
	}
}

func TestBuildNacosRegistryConfig(t *testing.T) {
	cfg := buildNacosRegistryConfig(nacosRegistrationOptions{
		NacosHost:      "nacos.local",
		NacosPort:      18848,
		NacosScheme:    "https",
		NacosNamespace: "dev",
		NacosGroup:     "MCP_GROUP",
		NacosCluster:   "blue",
	})
	if cfg.NamespaceID != "dev" || cfg.Group != "MCP_GROUP" {
		t.Fatalf("unexpected nacos cfg: %#v", cfg)
	}
	if len(cfg.Clusters) != 1 || cfg.Clusters[0] != "blue" {
		t.Fatalf("unexpected clusters: %#v", cfg.Clusters)
	}
	if len(cfg.Servers) != 1 || cfg.Servers[0].IP != "nacos.local" || cfg.Servers[0].Port != 18848 || cfg.Servers[0].Scheme != "https" {
		t.Fatalf("unexpected server cfg: %#v", cfg.Servers)
	}
}
