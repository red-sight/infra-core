package openapi

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"infra/registrator/internal/fsutil"
)

// specFetchTimeout bounds each service spec fetch so one hung service cannot
// stall aggregation.
const specFetchTimeout = 10 * time.Second

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
	hash    [32]byte
	http    *http.Client
}

// New creates an Aggregator with the given configuration.
func New(cfg Config) *Aggregator {
	return &Aggregator{cfg: cfg, http: &http.Client{Timeout: specFetchTimeout}}
}

// Aggregate fetches specs from all services, merges them, and writes to SpecsPath/openapi.json
// if the result differs from the last run. Returns true if the spec changed.
func (a *Aggregator) Aggregate(services []ServiceInfo) (bool, error) {
	spec := a.buildAggregated(services)

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return false, fmt.Errorf("openapi: marshal: %w", err)
	}

	h := sha256.Sum256(data)

	a.mu.Lock()
	changed := h != a.hash
	if changed {
		a.current = data
		a.hash = h
	}
	a.mu.Unlock()

	if !changed {
		return false, nil
	}

	path := filepath.Join(a.cfg.SpecsPath, "openapi.json")
	if err := fsutil.AtomicWrite(path, data, 0644); err != nil {
		return false, fmt.Errorf("openapi: write %s: %w", path, err)
	}
	return true, nil
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
	mergedScopes := map[string]interface{}{} // scope name → description
	var mergedTags []interface{}

	for _, svc := range services {
		raw, err := a.fetchSpec(svc)
		if err != nil {
			log.Printf("openapi: skip %s: %v", svc.Name, err)
			continue
		}
		paths, schemas, tags, scopes := a.processSpec(svc, raw)
		for k, v := range paths {
			mergedPaths[k] = v
		}
		for k, v := range schemas {
			mergedSchemas[k] = v
		}
		for k, v := range scopes {
			mergedScopes[k] = v
		}
		mergedTags = append(mergedTags, tags...)
	}

	if mergedTags == nil {
		mergedTags = []interface{}{}
	}

	oidcBase := fmt.Sprintf("%s://%s.%s", a.cfg.HTTPProtocol, a.cfg.OIDCSubdomain, a.cfg.BaseDomain)

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
				// oauth2 with authorizationCode lets Swagger UI show per-operation scope
				// requirements as checkboxes and list them on the lock icon popup.
				"BearerAuth": map[string]interface{}{
					"type": "oauth2",
					"flows": map[string]interface{}{
						"authorizationCode": map[string]interface{}{
							"authorizationUrl": oidcBase + "/oidc/auth",
							"tokenUrl":         oidcBase + "/oidc/token",
							"scopes":           mergedScopes,
						},
					},
				},
			},
			"schemas": mergedSchemas,
		},
	}
}

func (a *Aggregator) fetchSpec(svc ServiceInfo) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s:%d/%s", svc.Name, svc.Port, svc.OpenAPIRoute)
	resp, err := a.http.Get(url) //nolint:gosec
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

func (a *Aggregator) processSpec(svc ServiceInfo, raw map[string]interface{}) (paths, schemas map[string]interface{}, tags []interface{}, scopes map[string]interface{}) {
	version := specVersion(raw)
	prefix := fmt.Sprintf("/%s/%s/%s", a.cfg.APIRoute, svc.Name, version)

	rawPaths, _ := raw["paths"].(map[string]interface{})
	paths = make(map[string]interface{}, len(rawPaths))
	scopes = map[string]interface{}{}

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
			newOp, opScopes := processOperation(op, svc.AuthProtected)
			newItem[method] = newOp
			for _, s := range opScopes {
				scopes[s] = s // use scope name as its own description
			}
		}
		paths[prefix+rawPath] = newItem
	}

	if comps, ok := raw["components"].(map[string]interface{}); ok {
		if s, ok := comps["schemas"].(map[string]interface{}); ok {
			normalizeSchemas(s)
			schemas = s
		}
	}

	if rawTags, ok := raw["tags"].([]interface{}); ok {
		tags = rawTags
	}

	return paths, schemas, tags, scopes
}

// processOperation strips x-infra-* extensions, generates standard security fields,
// and returns the scopes it encountered.
func processOperation(op map[string]interface{}, defaultProtected bool) (map[string]interface{}, []string) {
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

	scopes := extractScopes(op)

	if protected {
		newOp["security"] = []interface{}{
			map[string]interface{}{"BearerAuth": scopes},
		}
	} else {
		// explicit empty array overrides any global security default
		newOp["security"] = []interface{}{}
	}

	return newOp, scopes
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

// normalizeSchemas converts OpenAPI 3.1 nullable type arrays to 3.0 nullable:true
// so that Swagger UI renders them correctly. Huma emits 3.1-style schemas
// ("type": ["array", "null"]) but the aggregated spec targets OpenAPI 3.0.
func normalizeSchemas(schemas map[string]interface{}) {
	for _, v := range schemas {
		if s, ok := v.(map[string]interface{}); ok {
			normalizeSchema(s)
		}
	}
}

func normalizeSchema(s map[string]interface{}) {
	// Convert ["X", "null"] → type: "X", nullable: true
	if types, ok := s["type"].([]interface{}); ok {
		nonNull := make([]string, 0, len(types))
		nullable := false
		for _, t := range types {
			if str, ok := t.(string); ok {
				if str == "null" {
					nullable = true
				} else {
					nonNull = append(nonNull, str)
				}
			}
		}
		if nullable && len(nonNull) == 1 {
			s["type"] = nonNull[0]
			s["nullable"] = true
		}
	}

	// Recurse into properties
	if props, ok := s["properties"].(map[string]interface{}); ok {
		for _, v := range props {
			if child, ok := v.(map[string]interface{}); ok {
				normalizeSchema(child)
			}
		}
	}

	// Recurse into items (arrays)
	if items, ok := s["items"].(map[string]interface{}); ok {
		normalizeSchema(items)
	}

	// Recurse into allOf / anyOf / oneOf
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if arr, ok := s[key].([]interface{}); ok {
			for _, v := range arr {
				if child, ok := v.(map[string]interface{}); ok {
					normalizeSchema(child)
				}
			}
		}
	}
}
