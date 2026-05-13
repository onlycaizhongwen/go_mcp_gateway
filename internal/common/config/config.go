package config

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/amoylab/unla/pkg/helper"
	"github.com/amoylab/unla/pkg/trace"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type (
	// SuperAdminConfig represents the super admin configuration
	SuperAdminConfig struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	}

	// ForwardConfig represents the forward configuration for header handling
	ForwardConfig struct {
		Enabled bool `yaml:"enabled"`
		McpArg  struct {
			KeyForHeader string `yaml:"key_for_header"`
		} `yaml:"mcp_arg"`
		Header struct {
			AllowHeaders     string `yaml:"allow_headers"`
			IgnoreHeaders    string `yaml:"ignore_headers"`
			CaseInsensitive  bool   `yaml:"case_insensitive"`
			OverrideExisting bool   `yaml:"override_existing"`
		} `yaml:"header"`
	}

	// MCPGatewayConfig represents the MCP gateway configuration
	MCPGatewayConfig struct {
		Port           int              `yaml:"port"`
		ReloadPort     int              `yaml:"reload_port"`
		ReloadInterval time.Duration    `yaml:"reload_interval"`
		ReloadSwitch   bool             `yaml:"reload_switch"`
		Forward        ForwardConfig    `yaml:"forward"`
		ToolAccess     ToolAccessConfig `yaml:"tool_access"`
		PID            string           `yaml:"pid"`
		SuperAdmin     SuperAdminConfig `yaml:"super_admin"`
		Logger         LoggerConfig     `yaml:"logger"`
		Tracing        trace.Config     `yaml:"tracing"`
		Storage        StorageConfig    `yaml:"storage"`
		Notifier       NotifierConfig   `yaml:"notifier"`
		Session        SessionConfig    `yaml:"session"`
		Auth           AuthConfig       `yaml:"auth"`
		Metrics        MetricsConfig    `yaml:"metrics"`
		Registry       RegistryConfig   `yaml:"registry"`
		Governance     GovernanceConfig `yaml:"governance"`
	}

	// RegistryConfig controls service registry/discovery integration.
	RegistryConfig struct {
		Type  string              `yaml:"type"` // empty or nacos
		Nacos NacosRegistryConfig `yaml:"nacos"`
	}

	// NacosRegistryConfig contains Nacos naming client settings.
	NacosRegistryConfig struct {
		NamespaceID     string              `yaml:"namespace_id"`
		Group           string              `yaml:"group"`
		Clusters        StringList          `yaml:"clusters"`
		Servers         []NacosServerConfig `yaml:"servers"`
		Username        string              `yaml:"username"`
		Password        string              `yaml:"password"`
		AccessKey       string              `yaml:"access_key"`
		SecretKey       string              `yaml:"secret_key"`
		TimeoutMS       uint64              `yaml:"timeout_ms"`
		CacheDir        string              `yaml:"cache_dir"`
		LogDir          string              `yaml:"log_dir"`
		RefreshInterval time.Duration       `yaml:"refresh_interval"`
	}

	// NacosServerConfig identifies a Nacos server endpoint.
	NacosServerConfig struct {
		IP      string `yaml:"ip"`
		Port    uint64 `yaml:"port"`
		Scheme  string `yaml:"scheme"`
		Context string `yaml:"context"`
	}

	// MetricsConfig controls Prometheus metrics exposure
	MetricsConfig struct {
		Enabled   bool      `yaml:"enabled"`
		Path      string    `yaml:"path"`
		Namespace string    `yaml:"namespace"`
		Buckets   []float64 `yaml:"buckets"`
	}

	// ToolAccessConfig controls access policies for tool endpoints.
	ToolAccessConfig struct {
		InternalNetwork InternalNetworkAccessConfig `yaml:"internal_network"`
	}

	// InternalNetworkAccessConfig defines allowlist for internal network targets.
	InternalNetworkAccessConfig struct {
		// Enabled controls whether internal network access validation is enabled.
		Enabled   *bool      `yaml:"enabled"`
		Allowlist StringList `yaml:"allowlist"`
	}

	// SessionConfig represents the session storage configuration
	SessionConfig struct {
		Type  string             `yaml:"type"`  // "memory" or "redis"
		Redis SessionRedisConfig `yaml:"redis"` // Redis configuration
	}

	// SessionRedisConfig represents the Redis configuration for session storage
	SessionRedisConfig struct {
		ClusterType string        `yaml:"cluster_type"` // "single", "cluster" or "sentinel"
		Addr        string        `yaml:"addr"`         // multiple addresses separated by ;.
		MasterName  string        `yaml:"master_name"`  // MasterName is the sentinel master name.
		Username    string        `yaml:"username"`
		Password    string        `yaml:"password"`
		DB          int           `yaml:"db"`
		Topic       string        `yaml:"topic"`
		Prefix      string        `yaml:"prefix"`
		TTL         time.Duration `yaml:"ttl"` // TTL for session data in Redis
	}

	// LoggerConfig represents the logger configuration
	LoggerConfig struct {
		Level      string `yaml:"level"`       // debug, info, warn, error
		Format     string `yaml:"format"`      // json, console
		Output     string `yaml:"output"`      // stdout, file
		FilePath   string `yaml:"file_path"`   // path to log file when output is file
		MaxSize    int    `yaml:"max_size"`    // max size of log file in MB
		MaxBackups int    `yaml:"max_backups"` // max number of backup files
		MaxAge     int    `yaml:"max_age"`     // max age of backup files in days
		Compress   bool   `yaml:"compress"`    // whether to compress backup files
		Color      bool   `yaml:"color"`       // whether to use color in console output
		Stacktrace bool   `yaml:"stacktrace"`  // whether to include stacktrace in error logs
		TimeZone   string `yaml:"time_zone"`   // time zone for log timestamps, e.g., "UTC", default is local
		TimeFormat string `yaml:"time_format"` // time format for log timestamps, default is "2006-01-02 15:04:05"
	}

	// AuthConfig defines the authentication configuration
	AuthConfig struct {
		OAuth2 *OAuth2Config      `yaml:"oauth2"`
		CORS   *CORSConfig        `yaml:"cors,omitempty"`
		Google *GoogleOAuthConfig `yaml:"google,omitempty"`
		GitHub *GitHubOAuthConfig `yaml:"github,omitempty"`
	}
	OAuth2Config struct {
		Issuer  string              `yaml:"issuer"`
		Storage OAuth2StorageConfig `yaml:"storage"`
	}
	OAuth2StorageConfig struct {
		Type  string            `yaml:"type"`
		Redis OAuth2RedisConfig `yaml:"redis"`
	}
	OAuth2RedisConfig struct {
		ClusterType string `yaml:"cluster_type"` // "single", "cluster" or "sentinel"
		Addr        string `yaml:"addr"`
		MasterName  string `yaml:"master_name"` // MasterName is the sentinel master name.
		Username    string `yaml:"username"`
		Password    string `yaml:"password"`
		DB          int    `yaml:"db"`
	}

	// GoogleOAuthConfig defines Google OAuth configuration
	GoogleOAuthConfig struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		RedirectURI  string `yaml:"redirect_uri"`
	}

	// GitHubOAuthConfig defines GitHub OAuth configuration
	GitHubOAuthConfig struct {
		ClientID     string `yaml:"client_id"`
		ClientSecret string `yaml:"client_secret"`
		RedirectURI  string `yaml:"redirect_uri"`
	}
)

type Type interface {
	MCPGatewayConfig | APIServerConfig
}

// StringList supports YAML sequences and comma-separated strings.
type StringList []string

func (s *StringList) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	if err := value.Decode(&raw); err == nil {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			*s = []string{}
			return nil
		}
		parts := strings.Split(raw, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		*s = out
		return nil
	}

	var items []string
	if err := value.Decode(&items); err != nil {
		return err
	}
	*s = items
	return nil
}

// LoadConfig loads configuration from a YAML file with environment variable support
func LoadConfig[T Type](filename string) (*T, string, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfgPath := helper.GetCfgPath(filename)
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, cfgPath, err
	}

	// Resolve environment variables
	data = resolveEnv(data)
	var cfg T
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, cfgPath, err
	}

	// Validate durations after unmarshalling
	if mcpCfg, ok := any(&cfg).(*MCPGatewayConfig); ok {
		if mcpCfg.ReloadInterval <= time.Second {
			mcpCfg.ReloadInterval = 600 * time.Second
		}
		applyMCPGatewayDefaults(mcpCfg)
		// Metrics defaults
		if mcpCfg.Metrics.Enabled {
			if mcpCfg.Metrics.Path == "" {
				mcpCfg.Metrics.Path = "/metrics"
			}
			if mcpCfg.Metrics.Namespace == "" {
				mcpCfg.Metrics.Namespace = "mcp_gateway"
			}
			if len(mcpCfg.Metrics.Buckets) == 0 {
				mcpCfg.Metrics.Buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0, 7.5, 10}
			}
		}

	}

	// Set defaults for apiserver MCP runtime if missing
	if apiCfg, ok := any(&cfg).(*APIServerConfig); ok {
		if apiCfg.MCP.CapabilitiesRefreshInterval <= 0 {
			apiCfg.MCP.CapabilitiesRefreshInterval = 120 * time.Second
		}
		if apiCfg.MCP.CapabilitiesCacheTTL <= 0 {
			apiCfg.MCP.CapabilitiesCacheTTL = 5 * time.Minute
		}
	}

	return &cfg, cfgPath, nil
}

func applyMCPGatewayDefaults(cfg *MCPGatewayConfig) {
	if cfg == nil {
		return
	}
	if cfg.Registry.Type == "nacos" {
		if cfg.Registry.Nacos.Group == "" {
			cfg.Registry.Nacos.Group = "DEFAULT_GROUP"
		}
		if len(cfg.Registry.Nacos.Clusters) == 0 {
			cfg.Registry.Nacos.Clusters = []string{"DEFAULT"}
		}
		if cfg.Registry.Nacos.TimeoutMS == 0 {
			cfg.Registry.Nacos.TimeoutMS = 5000
		}
		if cfg.Registry.Nacos.RefreshInterval <= 0 {
			cfg.Registry.Nacos.RefreshInterval = 10 * time.Second
		}
		for i := range cfg.Registry.Nacos.Servers {
			if cfg.Registry.Nacos.Servers[i].Scheme == "" {
				cfg.Registry.Nacos.Servers[i].Scheme = "http"
			}
		}
	}
	applyGovernanceDefaults(&cfg.Governance)
}

func applyGovernanceDefaults(cfg *GovernanceConfig) {
	if cfg == nil {
		return
	}
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

// resolveEnv replaces environment variable placeholders in YAML content
func resolveEnv(content []byte) []byte {
	regex := regexp.MustCompile(`\$\{(\w+)(?::([^}]*))?\}`)

	return regex.ReplaceAllFunc(content, func(match []byte) []byte {
		matches := regex.FindSubmatch(match)
		envKey := string(matches[1])
		var defaultValue string

		if len(matches) > 2 {
			defaultValue = string(matches[2])
		}

		if value, exists := os.LookupEnv(envKey); exists {
			return []byte(value)
		}
		return []byte(defaultValue)
	})
}
