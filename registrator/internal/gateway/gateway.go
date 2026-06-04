package gateway

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Config holds the runtime configuration for KrakenD config generation.
type Config struct {
	HTTPProtocol    string
	BaseDomain      string
	OIDCSubdomain   string
	APIRoute        string
	LogtoResourceID string
	ConfigPath      string // path to krakend.json
}

// ServiceInfo identifies a registered service and where to find its spec.
type ServiceInfo struct {
	Name          string
	Port          int
	OpenAPIRoute  string
	AuthProtected bool
}

// Generator reads service OpenAPI specs and writes a complete krakend.json.
type Generator struct {
	cfg  Config
	mu   sync.Mutex
	hash [32]byte
}

func New(cfg Config) *Generator {
	return &Generator{cfg: cfg}
}

// Generate fetches specs from all services, derives KrakenD endpoints, and rewrites ConfigPath
// if the result differs from the last run. Returns true if the config changed.
func (g *Generator) Generate(services []ServiceInfo) (bool, error) {
	var endpoints []interface{}
	for _, svc := range services {
		raw, err := g.fetchSpec(svc)
		if err != nil {
			log.Printf("gateway: skip %s: %v", svc.Name, err)
			continue
		}
		endpoints = append(endpoints, g.endpointsFromSpec(svc, raw)...)
	}

	data, err := g.marshalConfig(endpoints)
	if err != nil {
		return false, err
	}

	h := sha256.Sum256(data)

	g.mu.Lock()
	changed := h != g.hash
	if changed {
		g.hash = h
	}
	g.mu.Unlock()

	if !changed {
		return false, nil
	}

	return true, os.WriteFile(g.cfg.ConfigPath, data, 0644)
}

// --- spec fetching (mirrors openapi package — kept separate to avoid cross-package dependency) ---

func (g *Generator) fetchSpec(svc ServiceInfo) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s:%d/%s", svc.Name, svc.Port, svc.OpenAPIRoute)
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var spec map[string]interface{}
	return spec, json.Unmarshal(body, &spec)
}

// --- endpoint derivation ---

func (g *Generator) endpointsFromSpec(svc ServiceInfo, raw map[string]interface{}) []interface{} {
	version := specVersion(raw)
	prefix := fmt.Sprintf("/%s/%s/%s", g.cfg.APIRoute, svc.Name, version)
	host := fmt.Sprintf("http://%s:%d", svc.Name, svc.Port)

	var eps []interface{}
	rawPaths, _ := raw["paths"].(map[string]interface{})
	for rawPath, pathItem := range rawPaths {
		pathItemMap, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}
		for _, method := range httpMethods {
			op, ok := pathItemMap[method].(map[string]interface{})
			if !ok {
				continue
			}
			protected := svc.AuthProtected
			if v, ok := op["x-infra-protected"].(bool); ok {
				protected = v
			}
			ep := g.buildEndpoint(
				strings.ToUpper(method),
				prefix+rawPath,
				rawPath,
				host,
				protected,
				extractScopes(op),
				scopesMatcher(op),
			)
			eps = append(eps, ep)
		}
	}
	return eps
}

func (g *Generator) buildEndpoint(method, gatewayPath, backendPath, host string, protected bool, scopes []string, matcher string) map[string]interface{} {
	ep := map[string]interface{}{
		"endpoint":        gatewayPath,
		"method":          method,
		"output_encoding": "json",
		"backend": []map[string]interface{}{
			{
				"url_pattern": backendPath,
				"method":      method,
				"host":        []string{host},
				"encoding":    "json",
			},
		},
	}

	if !protected {
		return ep
	}

	jwkURL := fmt.Sprintf("%s://%s.%s/oidc/jwks",
		g.cfg.HTTPProtocol, g.cfg.OIDCSubdomain, g.cfg.BaseDomain)

	validator := map[string]interface{}{
		"alg":                  "RS256",
		"jwk_url":              jwkURL,
		"disable_jwk_security": g.cfg.HTTPProtocol == "http",
		"audience":             []string{g.cfg.LogtoResourceID},
		"propagate_claims": [][]string{
			{"sub", "x-user-id"},
			{"roles", "x-user-roles"},
			{"scope", "x-user-permissions"},
		},
	}
	if len(scopes) > 0 {
		validator["scopes"] = scopes
		validator["scopes_matcher"] = matcher
	}

	ep["extra_config"] = map[string]interface{}{
		"auth/validator": validator,
	}
	ep["input_headers"] = []string{"Authorization"}
	return ep
}

// --- config file I/O ---

// marshalConfig reads the base krakend.json, replaces the endpoints array, and returns the result.
// The base config (version, name, timeout, extra_config) is preserved as-is.
func (g *Generator) marshalConfig(endpoints []interface{}) ([]byte, error) {
	data, err := os.ReadFile(g.cfg.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("gateway: read %s: %w", g.cfg.ConfigPath, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("gateway: parse config: %w", err)
	}

	if endpoints == nil {
		endpoints = []interface{}{}
	}
	config["endpoints"] = endpoints

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("gateway: marshal config: %w", err)
	}
	return out, nil
}

// --- helpers ---

func specVersion(raw map[string]interface{}) string {
	if info, ok := raw["info"].(map[string]interface{}); ok {
		if v, ok := info["version"].(string); ok && v != "" {
			return v
		}
	}
	return "v1"
}

func extractScopes(op map[string]interface{}) []string {
	raw, ok := op["x-infra-scopes"].([]interface{})
	if !ok {
		return nil
	}
	scopes := make([]string, 0, len(raw))
	for _, s := range raw {
		if str, ok := s.(string); ok {
			scopes = append(scopes, str)
		}
	}
	return scopes
}

// scopesMatcher reads x-infra-scopes-matcher from the operation.
// Defaults to "any": multiple scopes are treated as alternatives (OR),
// which is the common case — different permissions granting the same access.
// Use "all" explicitly when ALL listed scopes must be present simultaneously.
func scopesMatcher(op map[string]interface{}) string {
	if v, ok := op["x-infra-scopes-matcher"].(string); ok && v == "all" {
		return "all"
	}
	return "any"
}

var httpMethods = []string{
	"get", "post", "put", "patch", "delete", "head", "options", "trace",
}
