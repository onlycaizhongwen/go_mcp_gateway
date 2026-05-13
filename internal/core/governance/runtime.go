package governance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/pkg/mcp"
)

var (
	ErrPermissionDenied = errors.New("permission denied")
	ErrRateLimited      = errors.New("rate limit exceeded")
	ErrCircuitOpen      = errors.New("circuit breaker is open")
)

type Invoker func(context.Context) (*mcp.CallToolResult, error)

type InvokeRequest struct {
	Context context.Context
	Header  http.Header
	Tenant  string
	Prefix  string
	Server  string
	Tool    string
	Policy  config.GovernanceConfig
}

type Runtime struct {
	mu       sync.Mutex
	limiters map[string]*tokenBucket
	breakers map[string]*circuitBreaker
}

func NewRuntime() *Runtime {
	return &Runtime{
		limiters: make(map[string]*tokenBucket),
		breakers: make(map[string]*circuitBreaker),
	}
}

func MergePolicy(global, local config.GovernanceConfig) config.GovernanceConfig {
	merged := global
	if local.Authorization.Enabled || local.Authorization.Mode != "" || len(local.Authorization.Rules) > 0 || local.Authorization.AllowByDefault {
		merged.Authorization = local.Authorization
	}
	if local.Timeout.Request > 0 {
		merged.Timeout.Request = local.Timeout.Request
	}
	if local.Timeout.Connect > 0 {
		merged.Timeout.Connect = local.Timeout.Connect
	}
	if local.RateLimit.Enabled || local.RateLimit.QPS > 0 || local.RateLimit.Burst > 0 || local.RateLimit.Dimension != "" {
		merged.RateLimit = local.RateLimit
	}
	if local.CircuitBreaker.Enabled || local.CircuitBreaker.MinRequests > 0 || local.CircuitBreaker.ErrorRate > 0 ||
		local.CircuitBreaker.OpenDuration > 0 || local.CircuitBreaker.HalfOpenMaxRequests > 0 {
		merged.CircuitBreaker = local.CircuitBreaker
	}
	if local.Fallback.Mode != "" || local.Fallback.Message != "" || local.Fallback.StaticText != "" || local.Fallback.ServiceName != "" {
		merged.Fallback = local.Fallback
	}
	applyDefaults(&merged)
	return merged
}

func (r *Runtime) Invoke(req InvokeRequest, invoke Invoker) (*mcp.CallToolResult, error) {
	if req.Context == nil {
		req.Context = context.Background()
	}
	applyDefaults(&req.Policy)

	if err := authorize(req); err != nil {
		return fallback(req.Policy.Fallback, err), nil
	}

	if err := r.allowRate(req); err != nil {
		return fallback(req.Policy.Fallback, err), nil
	}

	breaker, breakerKey := r.getBreaker(req)
	if breaker != nil {
		if err := breaker.before(); err != nil {
			return fallback(req.Policy.Fallback, fmt.Errorf("%w: %s", ErrCircuitOpen, breakerKey)), nil
		}
	}

	callCtx := req.Context
	cancel := func() {}
	if req.Policy.Timeout.Request > 0 {
		callCtx, cancel = context.WithTimeout(req.Context, req.Policy.Timeout.Request)
	}
	defer cancel()

	result, err := invoke(callCtx)
	if breaker != nil {
		breaker.after(err == nil)
	}
	if err != nil {
		return fallback(req.Policy.Fallback, err), nil
	}
	return result, nil
}

func applyDefaults(cfg *config.GovernanceConfig) {
	if cfg.Timeout.Request <= 0 {
		cfg.Timeout.Request = 30 * time.Second
	}
	if cfg.RateLimit.Burst <= 0 && cfg.RateLimit.QPS > 0 {
		cfg.RateLimit.Burst = cfg.RateLimit.QPS
	}
	if cfg.RateLimit.Dimension == "" {
		cfg.RateLimit.Dimension = "tenant_tool"
	}
	if cfg.CircuitBreaker.MinRequests <= 0 {
		cfg.CircuitBreaker.MinRequests = 20
	}
	if cfg.CircuitBreaker.ErrorRate <= 0 {
		cfg.CircuitBreaker.ErrorRate = 0.5
	}
	if cfg.CircuitBreaker.OpenDuration <= 0 {
		cfg.CircuitBreaker.OpenDuration = 30 * time.Second
	}
	if cfg.CircuitBreaker.HalfOpenMaxRequests <= 0 {
		cfg.CircuitBreaker.HalfOpenMaxRequests = 1
	}
	if cfg.Fallback.Mode == "" {
		cfg.Fallback.Mode = "mcp_error"
	}
}

func fallback(policy config.FallbackPolicy, cause error) *mcp.CallToolResult {
	msg := strings.TrimSpace(policy.Message)
	if msg == "" && cause != nil {
		msg = cause.Error()
	}
	if msg == "" {
		msg = "tool execution failed"
	}
	switch policy.Mode {
	case "static_text":
		text := policy.StaticText
		if text == "" {
			text = msg
		}
		return mcp.NewCallToolResultTextWithError(text, true)
	case "mcp_error", "":
		return mcp.NewCallToolResultError(msg)
	default:
		return mcp.NewCallToolResultError(msg)
	}
}
