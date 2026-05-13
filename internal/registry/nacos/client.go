package nacos

import (
	"context"
	"fmt"

	"github.com/amoylab/unla/internal/common/config"
	"github.com/amoylab/unla/internal/registry"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

type Client struct {
	naming naming_client.INamingClient
	cfg    config.NacosRegistryConfig
}

func NewClient(cfg config.NacosRegistryConfig) (*Client, error) {
	serverConfigs := make([]constant.ServerConfig, 0, len(cfg.Servers))
	for _, server := range cfg.Servers {
		if server.IP == "" || server.Port == 0 {
			continue
		}
		serverCfg := constant.ServerConfig{
			IpAddr:      server.IP,
			Port:        server.Port,
			Scheme:      server.Scheme,
			ContextPath: server.Context,
		}
		serverConfigs = append(serverConfigs, serverCfg)
	}
	if len(serverConfigs) == 0 {
		return nil, fmt.Errorf("nacos server list is empty")
	}

	clientConfig := constant.ClientConfig{
		NamespaceId: cfg.NamespaceID,
		TimeoutMs:   cfg.TimeoutMS,
		Username:    cfg.Username,
		Password:    cfg.Password,
		AccessKey:   cfg.AccessKey,
		SecretKey:   cfg.SecretKey,
		CacheDir:    cfg.CacheDir,
		LogDir:      cfg.LogDir,
	}
	naming, err := clients.NewNamingClient(vo.NacosClientParam{
		ClientConfig:  &clientConfig,
		ServerConfigs: serverConfigs,
	})
	if err != nil {
		return nil, err
	}
	return &Client{naming: naming, cfg: cfg}, nil
}

func (c *Client) ListHealthy(_ context.Context, ref registry.ServiceRef) ([]registry.Instance, error) {
	group := firstNonEmpty(ref.Group, c.cfg.Group, "DEFAULT_GROUP")
	clusters := ref.Clusters
	if len(clusters) == 0 {
		clusters = []string(c.cfg.Clusters)
	}
	if len(clusters) == 0 {
		clusters = []string{"DEFAULT"}
	}

	instances, err := c.naming.SelectInstances(vo.SelectInstancesParam{
		ServiceName: ref.ServiceName,
		GroupName:   group,
		Clusters:    clusters,
		HealthyOnly: ref.HealthyOnly,
	})
	if err != nil {
		return nil, err
	}

	out := make([]registry.Instance, 0, len(instances))
	for _, instance := range instances {
		converted := fromNacosInstance(ref.ServiceName, group, instance)
		if !matchTags(ref.Tags, converted.Metadata) {
			continue
		}
		out = append(out, converted)
	}
	return out, nil
}

func (c *Client) Register(_ context.Context, instance registry.Instance) error {
	group := firstNonEmpty(instance.Group, c.cfg.Group, "DEFAULT_GROUP")
	cluster := firstNonEmpty(instance.Cluster, "DEFAULT")
	ok, err := c.naming.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          instance.IP,
		Port:        instance.Port,
		ServiceName: instance.ServiceName,
		Weight:      instance.Weight,
		Enable:      instance.Enabled,
		Healthy:     instance.Healthy,
		Ephemeral:   true,
		Metadata:    instance.Metadata,
		ClusterName: cluster,
		GroupName:   group,
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("nacos register instance returned false")
	}
	return nil
}

func (c *Client) Deregister(_ context.Context, instance registry.Instance) error {
	group := firstNonEmpty(instance.Group, c.cfg.Group, "DEFAULT_GROUP")
	cluster := firstNonEmpty(instance.Cluster, "DEFAULT")
	ok, err := c.naming.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          instance.IP,
		Port:        instance.Port,
		ServiceName: instance.ServiceName,
		Ephemeral:   true,
		Cluster:     cluster,
		GroupName:   group,
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("nacos deregister instance returned false")
	}
	return nil
}

func fromNacosInstance(serviceName, group string, instance model.Instance) registry.Instance {
	return registry.Instance{
		ServiceName: serviceName,
		Group:       group,
		Cluster:     instance.ClusterName,
		IP:          instance.Ip,
		Port:        instance.Port,
		Weight:      instance.Weight,
		Healthy:     instance.Healthy,
		Enabled:     instance.Enable,
		Ephemeral:   instance.Ephemeral,
		Metadata:    instance.Metadata,
	}
}

func matchTags(expected map[string]string, actual map[string]string) bool {
	for k, v := range expected {
		if actual[k] != v {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
