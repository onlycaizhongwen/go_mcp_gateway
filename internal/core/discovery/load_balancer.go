package discovery

import (
	"math/rand"
	"sync"

	"github.com/amoylab/unla/internal/registry"
)

const (
	PolicyRoundRobin         = "round_robin"
	PolicyWeightedRoundRobin = "weighted_round_robin"
	PolicyRandom             = "random"
)

type LoadBalancer struct {
	policy string
	mu     sync.Mutex
	next   int
}

func NewLoadBalancer(policy string) *LoadBalancer {
	if policy == "" {
		policy = PolicyRoundRobin
	}
	return &LoadBalancer{policy: policy}
}

func (lb *LoadBalancer) Select(instances []registry.Instance) registry.Instance {
	if len(instances) == 0 {
		return registry.Instance{}
	}
	if lb == nil {
		return instances[0]
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	switch lb.policy {
	case PolicyRandom:
		return instances[rand.Intn(len(instances))]
	case PolicyWeightedRoundRobin:
		return lb.selectWeighted(instances)
	default:
		selected := instances[lb.next%len(instances)]
		lb.next++
		return selected
	}
}

func (lb *LoadBalancer) selectWeighted(instances []registry.Instance) registry.Instance {
	total := 0
	for _, instance := range instances {
		weight := int(instance.Weight)
		if weight <= 0 {
			weight = 1
		}
		total += weight
	}
	if total <= 0 {
		return instances[0]
	}

	slot := lb.next % total
	lb.next++
	cursor := 0
	for _, instance := range instances {
		weight := int(instance.Weight)
		if weight <= 0 {
			weight = 1
		}
		cursor += weight
		if slot < cursor {
			return instance
		}
	}
	return instances[len(instances)-1]
}
