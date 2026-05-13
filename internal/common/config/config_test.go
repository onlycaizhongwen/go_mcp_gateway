package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveEnv(t *testing.T) {
	t.Setenv("X_A", "va")
	in := []byte("a: ${X_A:da}\nb: ${X_B:db}")
	out := resolveEnv(in)
	assert.Contains(t, string(out), "a: va")
	assert.Contains(t, string(out), "b: db")
}

func TestLoadConfig_MCPGateway(t *testing.T) {
	tmp := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	_ = os.Chdir(tmp)

	// include env expansion and short reload interval to trigger defaulting
	yaml := `
port: 1234
reload_port: 0
reload_interval: 1s
reload_switch: true
forward:
  enabled: true
  mcp_arg:
    key_for_header: XH
  header:
    allow_headers: "A,B"
    ignore_headers: "C"
    case_insensitive: true
    override_existing: false
pid: ${X_PID:/tmp/gw.pid}
registry:
  type: nacos
  nacos:
    servers:
      - ip: 127.0.0.1
        port: 8848
governance:
  rate_limit:
    qps: 10
`
	file := filepath.Join(tmp, "mcp-gateway.yaml")
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o644))

	cfg, path, err := LoadConfig[MCPGatewayConfig]("mcp-gateway.yaml")
	assert.NoError(t, err)
	realFile, _ := filepath.EvalSymlinks(file)
	realPath, _ := filepath.EvalSymlinks(path)
	assert.Equal(t, realFile, realPath)
	assert.Equal(t, 1234, cfg.Port)
	// reload interval should be bumped to default (>= 600s)
	assert.GreaterOrEqual(t, int64(cfg.ReloadInterval), int64(600*time.Second))
	assert.True(t, cfg.Forward.Enabled)
	assert.Equal(t, "XH", cfg.Forward.McpArg.KeyForHeader)
	assert.Equal(t, "A,B", cfg.Forward.Header.AllowHeaders)
	assert.Equal(t, "C", cfg.Forward.Header.IgnoreHeaders)
	assert.True(t, cfg.Forward.Header.CaseInsensitive)
	assert.Equal(t, "nacos", cfg.Registry.Type)
	assert.Equal(t, "DEFAULT_GROUP", cfg.Registry.Nacos.Group)
	assert.Equal(t, []string{"DEFAULT"}, []string(cfg.Registry.Nacos.Clusters))
	assert.Equal(t, uint64(5000), cfg.Registry.Nacos.TimeoutMS)
	assert.Equal(t, 10*time.Second, cfg.Registry.Nacos.RefreshInterval)
	assert.Equal(t, "http", cfg.Registry.Nacos.Servers[0].Scheme)
	assert.Equal(t, 30*time.Second, cfg.Governance.Timeout.Request)
	assert.Equal(t, 10, cfg.Governance.RateLimit.Burst)
	assert.Equal(t, "tenant_tool", cfg.Governance.RateLimit.Dimension)
}

func TestLoadConfig_MCPGateway_InternalAllowlistString(t *testing.T) {
	tmp := t.TempDir()
	old, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(old) })
	_ = os.Chdir(tmp)

	yaml := `
tool_access:
  internal_network:
    allowlist: "127.0.0.1/32, localhost,,::1/128"
`
	file := filepath.Join(tmp, "mcp-gateway.yaml")
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o644))

	cfg, _, err := LoadConfig[MCPGatewayConfig]("mcp-gateway.yaml")
	assert.NoError(t, err)
	assert.Equal(t, []string{"127.0.0.1/32", "localhost", "::1/128"}, []string(cfg.ToolAccess.InternalNetwork.Allowlist))
}
