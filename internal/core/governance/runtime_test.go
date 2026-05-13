package governance

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/pkg/mcp"
)

func TestRuntimeAuthorizationDeniedUsesFallback(t *testing.T) {
	rt := NewRuntime()
	result, err := rt.Invoke(InvokeRequest{
		Context: context.Background(),
		Header:  http.Header{"X-Client-Id": []string{"client-a"}},
		Tenant:  "default",
		Server:  "svc",
		Tool:    "echo",
		Policy: config.GovernanceConfig{
			Authorization: config.AuthorizationPolicy{Enabled: true},
			Fallback:      config.FallbackPolicy{Mode: "mcp_error"},
		},
	}, func(ctx context.Context) (*mcp.CallToolResult, error) {
		t.Fatal("invoker should not be called")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Invoke returned unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected MCP error fallback, got %#v", result)
	}
}

func TestRuntimeTimeoutUsesFallback(t *testing.T) {
	rt := NewRuntime()
	result, err := rt.Invoke(InvokeRequest{
		Context: context.Background(),
		Policy: config.GovernanceConfig{
			Timeout:  config.TimeoutPolicy{Request: time.Millisecond},
			Fallback: config.FallbackPolicy{Mode: "static_text", StaticText: "timeout fallback"},
		},
	}, func(ctx context.Context) (*mcp.CallToolResult, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	if err != nil {
		t.Fatalf("Invoke returned unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected fallback result, got %#v", result)
	}
}

func TestRuntimeRateLimitUsesFallback(t *testing.T) {
	rt := NewRuntime()
	req := InvokeRequest{
		Context: context.Background(),
		Tenant:  "default",
		Server:  "svc",
		Tool:    "echo",
		Policy: config.GovernanceConfig{
			RateLimit: config.RateLimitPolicy{Enabled: true, QPS: 1, Burst: 1, Dimension: "tenant_tool"},
			Fallback:  config.FallbackPolicy{Mode: "mcp_error"},
		},
	}
	invoker := func(ctx context.Context) (*mcp.CallToolResult, error) {
		return mcp.NewCallToolResultText("ok"), nil
	}
	if _, err := rt.Invoke(req, invoker); err != nil {
		t.Fatalf("first invoke failed: %v", err)
	}
	result, err := rt.Invoke(req, invoker)
	if err != nil {
		t.Fatalf("second invoke returned unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected rate limit fallback, got %#v", result)
	}
}

func TestRuntimeCircuitBreakerOpens(t *testing.T) {
	rt := NewRuntime()
	req := InvokeRequest{
		Context: context.Background(),
		Tenant:  "default",
		Prefix:  "/demo",
		Server:  "svc",
		Tool:    "echo",
		Policy: config.GovernanceConfig{
			CircuitBreaker: config.CircuitBreakerPolicy{
				Enabled:             true,
				MinRequests:         1,
				ErrorRate:           0.5,
				OpenDuration:        time.Minute,
				HalfOpenMaxRequests: 1,
			},
			Fallback: config.FallbackPolicy{Mode: "mcp_error"},
		},
	}
	_, err := rt.Invoke(req, func(ctx context.Context) (*mcp.CallToolResult, error) {
		return nil, errors.New("backend failed")
	})
	if err != nil {
		t.Fatalf("first invoke returned unexpected error: %v", err)
	}
	result, err := rt.Invoke(req, func(ctx context.Context) (*mcp.CallToolResult, error) {
		t.Fatal("open breaker should block invocation")
		return mcp.NewCallToolResultText("ok"), nil
	})
	if err != nil {
		t.Fatalf("second invoke returned unexpected error: %v", err)
	}
	if result == nil || !result.IsError {
		t.Fatalf("expected circuit breaker fallback, got %#v", result)
	}
}
