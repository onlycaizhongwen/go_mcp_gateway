package governance

import (
	"fmt"
	"strings"
	"time"
)

func (r *Runtime) allowRate(req InvokeRequest) error {
	policy := req.Policy.RateLimit
	if !policy.Enabled || policy.QPS <= 0 {
		return nil
	}
	key := rateKey(req)
	r.mu.Lock()
	bucket := r.limiters[key]
	if bucket == nil || bucket.qps != policy.QPS || bucket.burst != policy.Burst {
		bucket = newTokenBucket(policy.QPS, policy.Burst)
		r.limiters[key] = bucket
	}
	allowed := bucket.allow(time.Now())
	r.mu.Unlock()
	if !allowed {
		return fmt.Errorf("%w: %s", ErrRateLimited, key)
	}
	return nil
}

func rateKey(req InvokeRequest) string {
	identity := identityFromHeader(req.Header)
	parts := []string{}
	for _, token := range strings.Split(req.Policy.RateLimit.Dimension, "_") {
		switch token {
		case "tenant":
			parts = append(parts, req.Tenant)
		case "prefix":
			parts = append(parts, req.Prefix)
		case "server":
			parts = append(parts, req.Server)
		case "tool":
			parts = append(parts, req.Tool)
		case "client":
			parts = append(parts, identity.clientID)
		}
	}
	if len(parts) == 0 {
		parts = append(parts, req.Tenant, req.Server, req.Tool)
	}
	return strings.Join(parts, "|")
}

type tokenBucket struct {
	qps    int
	burst  int
	tokens float64
	last   time.Time
}

func newTokenBucket(qps, burst int) *tokenBucket {
	if burst <= 0 {
		burst = qps
	}
	return &tokenBucket{
		qps:    qps,
		burst:  burst,
		tokens: float64(burst),
		last:   time.Now(),
	}
}

func (b *tokenBucket) allow(now time.Time) bool {
	elapsed := now.Sub(b.last).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * float64(b.qps)
		if b.tokens > float64(b.burst) {
			b.tokens = float64(b.burst)
		}
		b.last = now
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
