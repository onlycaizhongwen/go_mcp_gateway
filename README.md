
Go MCP Gateway | Enterprise Edition (企业级增强版)

Positioning | 项目定位  

An enterprise-grade MCP Gateway built on Nacos Ecosystem, solving MCP Server Discovery, Traffic Governance, and AI Tool Reliability.  

基于 Nacos 生态 构建的企业级 MCP 网关，专注于解决 AI 场景下 MCP Server 的动态发现、统一调度与流量治理 问题。

Based on the upstream mcp-ecosystem/mcpgateway, this project focuses on production readiness.  
本项目基于上游 mcpgateway 进行二次开发，核心目标是填补 AI 生产环境中的治理空白。

Why This Exists (痛点与破局)

In enterprise AI scenarios, MCP Servers are not static endpoints. They are:
在企业级 AI 落地场景中，MCP Server 不再是静态端点，而是：

• Dynamic Scaling (K8s / VM elasticity) | 动态伸缩 (容器与虚拟机弹性)

• Heterogeneous (Java Spring Cloud, Python FastAPI, Go Microservices) | 异构技术栈

• High Risk (LLM-induced high-frequency calls, unpredictable latency) | 高风险 (大模型引发的高频调用与抖动)

Standard MCP Gateways lack the following capabilities:  
标准 MCP 网关通常缺失以下能力：

1. Service Discovery | 服务发现 (No native support for Nacos/Eureka/Consul).
2. Traffic Governance | 流量治理 (No built-in rate limiting or circuit breaking for tools/call).
3. Stability | 稳定性 (No fallback mechanism when downstream fails).

Architecture | 架构总览

┌─────────────┐
│ MCP Client  │ (Claude / Cursor / Codex)
└──────┬──────┘
       │ SSE / Streamable HTTP
┌──────▼──────────┐
│  MCP Gateway   │◄──────┐
│  (Go)           │       │ Governance Chain
│                │       │ 1. AuthZ
└──────┬──────────┘       │ 2. Timeout
       │                  │ 3. Rate Limit
       │ Discovery        │ 4. Circuit Breaker
┌──────▼──────────┐       │ 5. Fallback
│ Nacos Registry  │───────┘
│ (Service Mesh)  │
└──────┬──────────┘
       │ Healthy Instances
┌──────▼──────────┐
│ MCP Server(s)   │
│ (Java/Py/Go)    │
└─────────────────┘


Core Features (核心能力)

1. Nacos-Centric Service Discovery

1. 以 Nacos 为核心的动态发现

Instead of static YAML configuration, MCP Servers register themselves to Nacos dynamically.
摒弃静态 YAML 配置，MCP Server 启动时自动向 Nacos 注册。

• Abstracted Registry Layer: Decoupled Discovery / Registrar interfaces.  

  抽象注册层：解耦 Discovery/Registrar 接口，支持未来扩展 Eureka/K8s。
• Metadata-Driven Protocols: Supports both SSE and Streamable HTTP.  

  元数据驱动：通过 Nacos Metadata 区分 SSE 与 Streamable HTTP 协议。

2. MCP-Aware Governance Chain (Critical)

2. 面向 MCP 的网关治理链（核心差异）

Governance is injected specifically into the tools/call lifecycle.
治理逻辑精准挂载在 tools/call 生命周期上，防止 AI Agent 拖垮后端系统。

Feature Implementation Purpose

Authorization Allowlist Mode 防止未授权的 Agent 调用敏感工具

Timeout Per-Tool Config 防止大模型阻塞等待

Rate Limit Token Bucket 保护遗留 Java 系统免受 LLM "Spam"

Circuit Breaker Error Rate Threshold 下游故障时快速熔断，防止雪崩

Fallback Static Text / Cache 优雅降级，返回友好提示而非 5xx

3. Dual Protocol Proxy

3. 双协议代理支持

• Full compatibility with MCP SSE.

• Support for Streamable HTTP (Replacement for long-polling).

Quick Start (Local Verification) | 快速开始

1. Start Nacos (Standalone)

1. 启动 Nacos 单机版

docker run -d --name nacos-standalone `
  -e MODE=standalone `
  -p 8848:8848 `
  nacos/nacos-server:v2.3.2


2. Run Mock MCP Server (Auto Register)

2. 启动 Mock MCP Server（自动注册）

go run ./cmd/mock-server `
  --register-nacos `
  --nacos-service-name mock-user-sse `
  --mcp-register-port 15337


3. Run Gateway

3. 启动网关

go run ./cmd/mcp-gateway --config config.yaml


Configuration Snippets | 配置示例

Global Registry (Nacos)

全局注册中心配置

registry:
  type: nacos
  nacos:
    namespace_id: public
    servers: [{ ip: 127.0.0.1, port: 8848 }]


Governance Policy (Safety First)

治理链配置（安全第一）

governance:
  rate_limit:
    enabled: true
    qps: 20      # Max 20 calls/sec per tool
  circuit_breaker:
    enabled: true
    error_rate: 0.5  # Open circuit if 50% calls fail
  fallback:
    enabled: true
    static_text: "System busy, please try again later."


Verification Status | 验证状态

✅ Nacos Auto-Registration / Deregistration | Nacos 自动注册与注销  
✅ Live Discovery & Routing | 实时发现与路由  
✅ Rate Limiting & Circuit Breaking | 限流与熔断  
✅ SSE & Streamable HTTP Proxy | 双协议代理  
✅ Docker Compose Ready | 容器化就绪  

Relation to Upstream | 与原项目关系

This project extends https://github.com/mcp-ecosystem/mcpgateway.  
While upstream provides the core MCP protocol handling, this edition adds Enterprise Service Mesh capabilities required for production AI systems.

本项目是对上游 mcpgateway 的企业级增强。上游负责 MCP 协议核心解析，本仓库负责补齐 企业级服务治理与发现 能力，使其满足生产级 AI 系统的严苛要求。

License

This project follows the upstream MIT License.  
本项目遵循上游 MIT 开源协议。