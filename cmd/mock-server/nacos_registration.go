package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/internal/registry"
	nacosregistry "github.com/amoylab/unla/internal/registry/nacos"
	"go.uber.org/zap"
)

type nacosRegistrationOptions struct {
	Enabled        bool
	NacosHost      string
	NacosPort      uint64
	NacosScheme    string
	NacosNamespace string
	NacosGroup     string
	NacosCluster   string
	ServiceName    string
	RegisterIP     string
	RegisterPort   uint64
	MCPScheme      string
	MCPProtocol    string
	MCPEndpoint    string
	MCPHost        string
}

func registerSSEMCPServer(ctx context.Context, opts nacosRegistrationOptions, listenAddr string) error {
	client, err := nacosregistry.NewClient(buildNacosRegistryConfig(opts))
	if err != nil {
		return err
	}
	instance, err := buildNacosInstance(opts, listenAddr)
	if err != nil {
		return err
	}
	if err := client.Register(ctx, instance); err != nil {
		return err
	}
	logger.Info("Registered mock MCP server to Nacos",
		zap.String("service", instance.ServiceName),
		zap.String("group", instance.Group),
		zap.String("cluster", instance.Cluster),
		zap.String("address", instance.Address()),
		zap.Any("metadata", instance.Metadata),
	)
	return nil
}

func deregisterSSEMCPServer(opts nacosRegistrationOptions, listenAddr string) {
	client, err := nacosregistry.NewClient(buildNacosRegistryConfig(opts))
	if err != nil {
		logger.Warn("Failed to create Nacos client for deregister", zap.Error(err))
		return
	}
	instance, err := buildNacosInstance(opts, listenAddr)
	if err != nil {
		logger.Warn("Failed to build Nacos instance for deregister", zap.Error(err))
		return
	}
	if err := client.Deregister(context.Background(), instance); err != nil {
		logger.Warn("Failed to deregister mock MCP server from Nacos", zap.Error(err))
		return
	}
	logger.Info("Deregistered mock MCP server from Nacos",
		zap.String("service", instance.ServiceName),
		zap.String("group", instance.Group),
		zap.String("address", instance.Address()),
	)
}

func buildNacosRegistryConfig(opts nacosRegistrationOptions) config.NacosRegistryConfig {
	group := firstNonEmpty(opts.NacosGroup, "DEFAULT_GROUP")
	cluster := firstNonEmpty(opts.NacosCluster, "DEFAULT")
	return config.NacosRegistryConfig{
		NamespaceID: opts.NacosNamespace,
		Group:       group,
		Clusters:    []string{cluster},
		TimeoutMS:   5000,
		CacheDir:    "./data/nacos/mock-server-cache",
		LogDir:      "./logs/nacos-mock-server",
		Servers: []config.NacosServerConfig{
			{
				IP:     firstNonEmpty(opts.NacosHost, "127.0.0.1"),
				Port:   firstNonZero(opts.NacosPort, 8848),
				Scheme: firstNonEmpty(opts.NacosScheme, "http"),
			},
		},
	}
}

func buildNacosInstance(opts nacosRegistrationOptions, listenAddr string) (registry.Instance, error) {
	port := opts.RegisterPort
	if port == 0 {
		parsedPort, err := portFromListenAddr(listenAddr)
		if err != nil {
			return registry.Instance{}, err
		}
		port = parsedPort
	}
	metadata := map[string]string{
		"mcp.endpoint": firstNonEmpty(opts.MCPEndpoint, "/sse"),
		"mcp.protocol": firstNonEmpty(opts.MCPProtocol, "sse"),
		"mcp.scheme":   firstNonEmpty(opts.MCPScheme, "http"),
	}
	if opts.MCPHost != "" {
		metadata["mcp.host"] = opts.MCPHost
	}
	return registry.Instance{
		ServiceName: firstNonEmpty(opts.ServiceName, "mock-user-sse"),
		Group:       firstNonEmpty(opts.NacosGroup, "DEFAULT_GROUP"),
		Cluster:     firstNonEmpty(opts.NacosCluster, "DEFAULT"),
		IP:          firstNonEmpty(opts.RegisterIP, "127.0.0.1"),
		Port:        port,
		Weight:      1,
		Healthy:     true,
		Enabled:     true,
		Metadata:    metadata,
	}, nil
}

func portFromListenAddr(addr string) (uint64, error) {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	_, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("parse listen address %q: %w", addr, err)
	}
	port, err := strconv.ParseUint(portText, 10, 64)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("invalid listen port %q", portText)
	}
	return port, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZero(values ...uint64) uint64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
