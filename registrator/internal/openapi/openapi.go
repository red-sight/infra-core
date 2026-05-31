package openapi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// Config holds the runtime configuration for the aggregator.
type Config struct {
	HTTPProtocol  string
	BaseDomain    string
	OIDCSubdomain string
	APIRoute      string
	SpecsPath     string
}

// ServiceInfo identifies a registered service and where to find its spec.
type ServiceInfo struct {
	Name          string
	Port          int
	OpenAPIRoute  string
	AuthProtected bool // default from infra.auth.protected Docker label
}

// Aggregator fetches, rewrites, and merges OpenAPI specs from registered services.
type Aggregator struct {
	cfg     Config
	mu      sync.RWMutex
	current []byte
}

// New creates an Aggregator with the given configuration.
func New(cfg Config) *Aggregator {
	return &Aggregator{cfg: cfg}
}

// Aggregate fetches specs from all services, merges them, writes to SpecsPath/openapi.json,
// and caches the result for the HTTP handler.
func (a *Aggregator) Aggregate(services []ServiceInfo) error {
	spec := a.buildAggregated(services)

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return fmt.Errorf("openapi: marshal: %w", err)
	}

	path := filepath.Join(a.cfg.SpecsPath, "openapi.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("openapi: write %s: %w", path, err)
	}

	a.mu.Lock()
	a.current = data
	a.mu.Unlock()
	return nil
}

// Handler returns an http.Handler serving the current aggregated spec at /openapi.json.
func (a *Aggregator) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		data := a.current
		a.mu.RUnlock()

		if data == nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}

func (a *Aggregator) buildAggregated(services []ServiceInfo) map[string]interface{} {
	mergedPaths := map[string]interface{}{}
	mergedSchemas := map[string]interface{}{}
	var mergedTags []interface{}

	for _, svc := range services {
		raw, err := a.fetchSpec(svc)
		if err != nil {
			log.Printf("openapi: skip %s: %v", svc.Name, err)
			continue
		}
		paths, schemas, tags := a.processSpec(svc, raw)
		for k, v := range paths {
			mergedPaths[k] = v
		}
		for k, v := range schemas {
			mergedSchemas[k] = v
		}
		mergedTags = append(mergedTags, tags...)
	}

	if mergedTags == nil {
		mergedTags = []interface{}{}
	}

	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Infra API",
			"version": "v1",
		},
		"servers": []map[string]interface{}{
			{"url": fmt.Sprintf("%s://%s", a.cfg.HTTPProtocol, a.cfg.BaseDomain)},
		},
		"tags":  mergedTags,
		"paths": mergedPaths,
		"components": map[string]interface{}{
			"securitySchemes": map[string]interface{}{
				"BearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT (Logto)",
				},
			},
			"schemas": mergedSchemas,
		},
	}
}

func (a *Aggregator) fetchSpec(svc ServiceInfo) (map[string]interface{}, error) {
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
	if err := json.Unmarshal(body, &spec); err != nil {
		return nil, fmt.Errorf("invalid JSON from %s: %w", url, err)
	}
	return spec, nil
}

func (a *Aggregator) processSpec(svc ServiceInfo, raw map[string]interface{}) (paths, schemas map[string]interface{}, tags []interface{}) {
	version := specVersion(raw)
	prefix := fmt.Sprintf("/%s/%s/%s", a.cfg.APIRoute, svc.Name, version)

	rawPaths, _ := raw["paths"].(map[string]interface{})
	paths = make(map[string]interface{}, len(rawPaths))

	for rawPath, pathItem := range rawPaths {
		pathItemMap, ok := pathItem.(map[string]interface{})
		if !ok {
			paths[prefix+rawPath] = pathItem
			continue
		}

		newItem := make(map[string]interface{}, len(pathItemMap))
		for k, v := range pathItemMap {
			newItem[k] = v
		}
		for _, method := range httpMethods {
			op, ok := newItem[method].(map[string]interface{})
			if !ok {
				continue
			}
			newItem[method] = processOperation(op, svc.AuthProtected)
		}
		paths[prefix+rawPath] = newItem
	}

	if comps, ok := raw["components"].(map[string]interface{}); ok {
		if s, ok := comps["schemas"].(map[string]interface{}); ok {
			schemas = s
		}
	}

	if rawTags, ok := raw["tags"].([]interface{}); ok {
		tags = rawTags
	}

	return paths, schemas, tags
}

// processOperation strips x-infra-* extensions and generates standard security fields.
func processOperation(op map[string]interface{}, defaultProtected bool) map[string]interface{} {
	newOp := make(map[string]interface{}, len(op))
	for k, v := range op {
		switch k {
		case "x-infra-protected", "x-infra-scopes", "x-infra-scopes-matcher":
			// converted to standard security fields below
		default:
			newOp[k] = v
		}
	}

	protected := defaultProtected
	if v, ok := op["x-infra-protected"].(bool); ok {
		protected = v
	}

	if protected {
		newOp["security"] = []interface{}{
			map[string]interface{}{"BearerAuth": extractScopes(op)},
		}
	} else {
		// explicit empty array overrides any global security default
		newOp["security"] = []interface{}{}
	}

	return newOp
}

func extractScopes(op map[string]interface{}) []string {
	raw, ok := op["x-infra-scopes"].([]interface{})
	if !ok {
		return []string{}
	}
	scopes := make([]string, 0, len(raw))
	for _, s := range raw {
		if str, ok := s.(string); ok {
			scopes = append(scopes, str)
		}
	}
	return scopes
}

func specVersion(raw map[string]interface{}) string {
	if info, ok := raw["info"].(map[string]interface{}); ok {
		if v, ok := info["version"].(string); ok && v != "" {
			return v
		}
	}
	return "v1"
}

var httpMethods = []string{
	"get", "post", "put", "patch", "delete", "head", "options", "trace",
}
