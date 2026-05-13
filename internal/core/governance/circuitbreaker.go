package governance

import (
	"strings"
	"sync"
	"time"

	"github.com/amoylab/unla/internal/common/config"
)

type circuitState string

const (
	circuitClosed   circuitState = "closed"
	circuitOpen     circuitState = "open"
	circuitHalfOpen circuitState = "half_open"
)

func (r *Runtime) getBreaker(req InvokeRequest) (*circuitBreaker, string) {
	policy := req.Policy.CircuitBreaker
	if !policy.Enabled {
		return nil, ""
	}
	key := strings.Join([]string{req.Tenant, req.Prefix, req.Server, req.Tool}, "|")
	r.mu.Lock()
	defer r.mu.Unlock()
	breaker := r.breakers[key]
	if breaker == nil || !sameCircuitPolicy(breaker.policy, policy) {
		breaker = newCircuitBreaker(policy)
		r.breakers[key] = breaker
	}
	return breaker, key
}

func sameCircuitPolicy(a, b config.CircuitBreakerPolicy) bool {
	return a.MinRequests == b.MinRequests &&
		a.ErrorRate == b.ErrorRate &&
		a.OpenDuration == b.OpenDuration &&
		a.HalfOpenMaxRequests == b.HalfOpenMaxRequests
}

type circuitBreaker struct {
	mu               sync.Mutex
	policy           config.CircuitBreakerPolicy
	state            circuitState
	openedUntil      time.Time
	requests         int
	failures         int
	halfOpenRequests int
}

func newCircuitBreaker(policy config.CircuitBreakerPolicy) *circuitBreaker {
	return &circuitBreaker{
		policy: policy,
		state:  circuitClosed,
	}
}

func (b *circuitBreaker) before() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if b.state == circuitOpen {
		if now.Before(b.openedUntil) {
			return ErrCircuitOpen
		}
		b.state = circuitHalfOpen
		b.halfOpenRequests = 0
	}
	if b.state == circuitHalfOpen {
		if b.halfOpenRequests >= b.policy.HalfOpenMaxRequests {
			return ErrCircuitOpen
		}
		b.halfOpenRequests++
	}
	return nil
}

func (b *circuitBreaker) after(success bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.state == circuitHalfOpen {
		if success {
			b.close()
			return
		}
		b.open()
		return
	}
	if b.state == circuitOpen {
		return
	}

	b.requests++
	if !success {
		b.failures++
	}
	if b.requests < b.policy.MinRequests {
		return
	}
	errorRate := float64(b.failures) / float64(b.requests)
	if errorRate >= b.policy.ErrorRate {
		b.open()
	}
}

func (b *circuitBreaker) open() {
	b.state = circuitOpen
	b.openedUntil = time.Now().Add(b.policy.OpenDuration)
	b.requests = 0
	b.failures = 0
	b.halfOpenRequests = 0
}

func (b *circuitBreaker) close() {
	b.state = circuitClosed
	b.openedUntil = time.Time{}
	b.requests = 0
	b.failures = 0
	b.halfOpenRequests = 0
}
