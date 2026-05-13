package governance

import (
	"fmt"
	"net/http"
	"strings"
)

func authorize(req InvokeRequest) error {
	policy := req.Policy.Authorization
	if !policy.Enabled {
		return nil
	}
	identity := identityFromHeader(req.Header)
	for _, rule := range policy.Rules {
		if matchAll(rule.Tenants, req.Tenant) &&
			matchAll(rule.Servers, req.Server) &&
			matchAll(rule.Tools, req.Tool) &&
			matchAll(rule.Clients, identity.clientID) &&
			matchAny(rule.Roles, identity.roles) &&
			matchAny(rule.Scopes, identity.scopes) {
			return nil
		}
	}
	if policy.AllowByDefault {
		return nil
	}
	return fmt.Errorf("%w: client=%s tenant=%s server=%s tool=%s", ErrPermissionDenied, identity.clientID, req.Tenant, req.Server, req.Tool)
}

type identity struct {
	clientID string
	roles    []string
	scopes   []string
}

func identityFromHeader(header http.Header) identity {
	clientID := firstHeader(header, "X-Client-Id", "Client-Id", "X-MCP-Client-Id")
	if clientID == "" {
		clientID = "anonymous"
	}
	return identity{
		clientID: clientID,
		roles:    splitCSV(firstHeader(header, "X-MCP-Roles", "X-Roles")),
		scopes:   splitCSV(firstHeader(header, "X-MCP-Scopes", "X-Scopes", "Scope")),
	}
}

func firstHeader(header http.Header, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(header.Get(key)); value != "" {
			return value
		}
	}
	return ""
}

func splitCSV(value string) []string {
	value = strings.ReplaceAll(value, " ", ",")
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func matchAll(patterns []string, value string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pattern := range patterns {
		if matchValue(pattern, value) {
			return true
		}
	}
	return false
}

func matchAny(required []string, actual []string) bool {
	if len(required) == 0 {
		return true
	}
	for _, req := range required {
		for _, got := range actual {
			if matchValue(req, got) {
				return true
			}
		}
	}
	return false
}

func matchValue(pattern, value string) bool {
	pattern = strings.TrimSpace(pattern)
	value = strings.TrimSpace(value)
	return pattern == "*" || strings.EqualFold(pattern, value)
}
