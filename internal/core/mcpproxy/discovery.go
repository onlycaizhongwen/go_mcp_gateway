package mcpproxy

import (
	"context"
	"fmt"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/internal/core/discovery"
	"github.com/amoylab/unla/internal/registry"
	"github.com/amoylab/unla/internal/template"
	"github.com/amoylab/unla/pkg/mcp"
)

// DiscoveryTransport resolves a concrete MCP server instance before each call.
type DiscoveryTransport struct {
	cfg     config.MCPServerConfig
	backend *discovery.Backend
}

var _ Transport = (*DiscoveryTransport)(nil)

func NewDiscoveryTransport(client registry.Discovery, cfg config.MCPServerConfig) (*DiscoveryTransport, error) {
	if client == nil {
		return nil, fmt.Errorf("discovery client is not configured")
	}
	return &DiscoveryTransport{
		cfg:     cfg,
		backend: discovery.NewBackend(client, cfg.Discovery, cfg.LoadBalance, cfg.Type),
	}, nil
}

func (t *DiscoveryTransport) FetchTools(ctx context.Context) ([]mcp.ToolSchema, error) {
	transport, err := t.resolve(ctx)
	if err != nil {
		return nil, err
	}
	defer transport.Stop(ctx)
	return transport.FetchTools(ctx)
}

func (t *DiscoveryTransport) CallTool(ctx context.Context, params mcp.CallToolParams, req *template.RequestWrapper) (*mcp.CallToolResult, error) {
	transport, err := t.resolve(ctx)
	if err != nil {
		return nil, err
	}
	defer transport.Stop(ctx)
	return transport.CallTool(ctx, params, req)
}

func (t *DiscoveryTransport) Start(context.Context, *template.Context) error {
	return nil
}

func (t *DiscoveryTransport) Stop(context.Context) error {
	return nil
}

func (t *DiscoveryTransport) IsRunning() bool {
	return false
}

func (t *DiscoveryTransport) FetchPrompts(ctx context.Context) ([]mcp.PromptSchema, error) {
	transport, err := t.resolve(ctx)
	if err != nil {
		return nil, err
	}
	defer transport.Stop(ctx)
	return transport.FetchPrompts(ctx)
}

func (t *DiscoveryTransport) FetchPrompt(ctx context.Context, name string) (*mcp.PromptSchema, error) {
	transport, err := t.resolve(ctx)
	if err != nil {
		return nil, err
	}
	defer transport.Stop(ctx)
	return transport.FetchPrompt(ctx, name)
}

func (t *DiscoveryTransport) resolve(ctx context.Context) (Transport, error) {
	selected, err := t.backend.Select(ctx)
	if err != nil {
		return nil, err
	}
	cfg := t.cfg
	cfg.URL = selected.URL
	cfg.Discovery.Enabled = false
	if selected.Protocol != "" {
		cfg.Type = selected.Protocol
	}
	return NewTransport(cfg)
}
