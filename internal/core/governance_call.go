package core

import (
	"context"

	"github.com/amoylab/unla/internal/core/governance"
	"github.com/amoylab/unla/internal/core/mcpproxy"
	"github.com/amoylab/unla/internal/mcp/session"
	"github.com/amoylab/unla/internal/template"
	"github.com/amoylab/unla/pkg/mcp"
	"github.com/gin-gonic/gin"
)

func (s *Server) callMCPServerToolWithGovernance(
	c *gin.Context,
	conn session.Connection,
	params mcp.CallToolParams,
	transport mcpproxy.Transport,
	req *template.RequestWrapper,
) (*mcp.CallToolResult, error) {
	prefix := conn.Meta().Prefix
	if s.governanceRuntime == nil {
		s.governanceRuntime = governance.NewRuntime()
	}
	serverCfg := s.state.GetMCPServerConfig(prefix)
	policy := s.governanceConfig
	serverName := ""
	if serverCfg != nil {
		policy = governance.MergePolicy(s.governanceConfig, serverCfg.Governance)
		serverName = serverCfg.Name
	}
	invokeReq := governance.InvokeRequest{
		Context: c.Request.Context(),
		Header:  c.Request.Header,
		Tenant:  s.state.GetTenant(prefix),
		Prefix:  prefix,
		Server:  serverName,
		Tool:    params.Name,
		Policy:  policy,
	}
	return s.governanceRuntime.Invoke(invokeReq, func(ctx context.Context) (*mcp.CallToolResult, error) {
		return transport.CallTool(ctx, params, req)
	})
}
