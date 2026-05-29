# Go MCP Gateway 二开项目 / Nacos Service Discovery Gateway

> 基于 Unla / MCP Gateway 的二次改造项目，面向企业级 MCP Server 接入、Nacos 服务发现、统一网关调度与治理链落地。  
> A customized Go-based MCP Gateway with Nacos service discovery, dynamic MCP server routing, and gateway governance.

## 项目定位 / Project Overview

本项目在开源 MCP Gateway 能力基础上进行二次开发，重点解决 MCP Server 在企业环境中的注册发现、统一调度、权限控制和稳定性治理问题。

核心目标：

- MCP Server 自动注册到 Nacos / automatic MCP server registration to Nacos
- Gateway 通过 Nacos 自动发现服务 / service discovery through Nacos
- 对 MCP 工具调用统一调度 / centralized MCP tool invocation routing
- 网关治理链：权限、超时、限流、熔断、降级
- 支持 SSE 与 Streamable HTTP MCP Server 代理
- 真实 Docker Nacos 环境联调验证


## 二开亮点 / Custom Features

### 1. Nacos 注册中心接入 / Nacos Registry Integration

- 新增统一注册中心抽象：`Discovery`、`Registrar`、`Client`
- 新增 Nacos SDK 适配层
- 支持健康实例查询、服务注册、服务注销
- 支持 namespace、group、cluster、metadata 配置

相关代码：

- `internal/registry`
- `internal/registry/nacos`
- `internal/common/config`

### 2. MCP Server 自动注册 / MCP Server Auto Registration

`cmd/mock-server` 已支持启动时自动注册 SSE MCP Server 到 Nacos，退出时自动注销。

示例：

```powershell
go run ./cmd/mock-server `
  --addr :15336 `
  --sse-addr :15337 `
  --register-nacos `
  --nacos-service-name mock-user-sse `
  --mcp-register-host 127.0.0.1 `
  --mcp-register-port 15337 `
  --mcp-host localhost
```

Nacos metadata 约定：

```text
mcp.endpoint=/sse
mcp.protocol=sse
mcp.scheme=http
mcp.host=localhost
```

### 3. 网关自动发现与统一调度 / Gateway Discovery Routing

网关启动时会根据配置创建 Nacos discovery client，并注入到核心服务。对于开启 discovery 的 MCP Server，网关会在每次调用前解析真实实例地址，再创建对应 transport 完成调用。

能力包括：

- DiscoveryTransport 动态解析实例
- 支持 round_robin、weighted_round_robin、random 负载均衡
- 支持通过 metadata 覆盖 endpoint、protocol、scheme、host
- 支持 SSE / Streamable HTTP 代理路径

相关代码：

- `internal/core/mcpproxy/discovery.go`
- `internal/core/discovery`
- `internal/core/state/state.go`
- `cmd/mcp-gateway/main.go`

### 4. 网关治理链 / Gateway Governance Chain

已在 MCP Server `tools/call` 调用路径接入治理能力：

- Authorization / 权限控制
- Request Timeout / 请求超时
- Local Rate Limit / 本地令牌桶限流
- Circuit Breaker / 错误率熔断
- Fallback / MCP 错误或静态文本降级

相关代码：

- `internal/core/governance`
- `internal/core/governance_call.go`
- `internal/core/server.go`

## 架构说明 / Architecture

```text
MCP Client
    |
    | SSE / Streamable HTTP
    v
MCP Gateway
    |
    | Authorization / Timeout / RateLimit / CircuitBreaker / Fallback
    v
DiscoveryTransport
    |
    | serviceName + group + cluster + metadata
    v
Nacos Registry
    |
    | healthy instances
    v
MCP Server Instance
```

## 用户视角调用流程 / User Flow

1. MCP Server 启动后自动注册到 Nacos。
2. 用户或 MCP Client 请求 Gateway 的 MCP endpoint。
3. Gateway 根据配置找到对应 MCP Server。
4. 如果该服务开启 discovery，Gateway 从 Nacos 获取健康实例。
5. Gateway 按负载均衡策略选择实例。
6. 调用进入治理链，依次执行权限、超时、限流、熔断和降级逻辑。
7. Gateway 转发到真实 MCP Server，并把工具调用结果返回给 MCP Client。

## 配置示例 / Config Example

全局注册中心：

```yaml
registry:
  type: nacos
  nacos:
    namespace_id: ""
    group: DEFAULT_GROUP
    clusters: DEFAULT
    servers:
      - ip: 127.0.0.1
        port: 8848
        scheme: http
```

MCP Server discovery：

```yaml
mcpServers:
  - name: mock-user-sse
    type: sse
    policy: onDemand
    discovery:
      enabled: true
      registry: nacos
      service_name: mock-user-sse
      group: DEFAULT_GROUP
      healthy_only: true
    load_balance:
      policy: round_robin
```

治理链：

```yaml
governance:
  authorization:
    enabled: true
    mode: allowlist
    allow_by_default: false
  timeout:
    request: 5s
  rate_limit:
    enabled: true
    qps: 20
    burst: 40
  circuit_breaker:
    enabled: true
    min_requests: 10
    error_rate: 0.5
    open_duration: 30s
  fallback:
    enabled: true
    mode: static_text
    static_text: "service temporarily unavailable"
```

## 本地验证 / Local Verification

启动 Nacos：

```powershell
docker run -d --name nacos-standalone `
  -e MODE=standalone `
  -p 8848:8848 `
  -p 9848:9848 `
  -p 9849:9849 `
  nacos/nacos-server:v2.3.2
```

定向测试：

```powershell
go test ./cmd/mock-server ./internal/core/discovery ./internal/core/mcpproxy ./internal/core/state ./internal/core ./internal/common/config ./internal/registry/...
```

真实 Nacos discovery 测试：

```powershell
$env:UNLA_LIVE_NACOS_TEST='1'
go test ./internal/core/mcpproxy -run TestDiscoveryTransportWithLiveNacos -v
```

自动注册链路测试：

```powershell
$env:UNLA_LIVE_NACOS_AUTOREG_TEST='1'
$env:UNLA_AUTOREG_SERVICE_NAME='mock-user-sse-autoreg-15337'
go test ./internal/core/mcpproxy -run TestDiscoveryTransportWithLiveNacosAutoRegisteredService -v
```

## 已验证能力 / Verified Capabilities

- Docker Nacos standalone 可用
- MCP Server 可自动注册到 Nacos
- MCP Server 停止后可从 Nacos 注销
- Gateway 可从 Nacos 发现健康实例
- Gateway 可通过发现实例调用 MCP tool
- 治理链覆盖 MCP Server `tools/call`
- 相关包定向测试通过

## 已知限制 / Known Limitations

- 当前自动注册示例主要覆盖 SSE MCP Server。
- Windows 下全量 `go test ./...` 仍受上游既有测试影响，包括临时文件锁和 Unix-only signal 测试；本次改造相关包已完成定向验证。

## 原项目说明 / Upstream

本项目基于 Unla / MCP Gateway 二次开发。原项目能力包括：

- REST API 转 MCP Server
- MCP Server proxy
- MCP SSE
- MCP Streamable HTTP
- 配置持久化与热更新
- Web 管理界面
- Docker / Kubernetes / Helm 部署

Upstream repository: `mcp-ecosystem/mcp-gateway`

## License

This project follows the upstream MIT License.
