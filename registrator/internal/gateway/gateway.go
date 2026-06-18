package gateway

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"infra/registrator/internal/fsutil"
	"infra/registrator/internal/versiongate"
)

// Config holds the runtime configuration for KrakenD config generation.
type Config struct {
	HTTPProtocol    string
	BaseDomain      string
	OIDCSubdomain   string
	APIRoute        string
	LogtoResourceID string
	ConfigPath      string // path to krakend.json

	// CORSAllowOrigins are the browser origins allowed to call the gateway (the
	// admin SPA lives on a different subdomain than the API). Empty disables CORS.
	CORSAllowOrigins []string
}

// ServiceInfo identifies a registered service and where to find its spec.
type ServiceInfo struct {
	Name          string
	Port          int
	OpenAPIRoute  string
	AuthProtected bool
}

// specFetchTimeout bounds each service spec fetch so one hung service cannot
// stall the whole reload cycle.
const specFetchTimeout = 10 * time.Second

// Generator reads service OpenAPI specs and writes a complete krakend.json.
type Generator struct {
	cfg      Config
	mu       sync.Mutex
	hash     [32]byte
	validate func(candidatePath string) error // optional; run on the candidate before promoting it
	http     *http.Client
}

func New(cfg Config) *Generator {
	return &Generator{cfg: cfg, http: &http.Client{Timeout: specFetchTimeout}}
}

// SetValidator installs a validation hook run against the freshly written
// candidate config before it is promoted over the live config. If it returns an
// error the candidate is discarded and the live config is left untouched. Used to
// run `krakend check` so an invalid config never reaches KrakenD.
func (g *Generator) SetValidator(fn func(candidatePath string) error) {
	g.validate = fn
}

// Generate fetches specs from all services, derives KrakenD endpoints, and rewrites ConfigPath
// if the result differs from the last run. Returns true if the config changed.
// scopeRoles maps scope name → role names that hold that scope; nil means roles are unavailable
// (Logto init not yet complete) and the generator falls back to scope-based validation.
func (g *Generator) Generate(services []ServiceInfo, scopeRoles map[string][]string) (bool, error) {
	data, endpoints, _, err := g.build(services, scopeRoles)
	if err != nil {
		return false, err
	}

	h := sha256.Sum256(data)

	g.mu.Lock()
	unchanged := h == g.hash
	g.mu.Unlock()
	if unchanged {
		return false, nil
	}

	logDenyAllEndpoints(endpoints)

	// Write the candidate, validate it (e.g. `krakend check`), and only then
	// promote it over the live config. On failure the hash is left unchanged so
	// the next pass retries rather than treating the bad config as delivered.
	if err := fsutil.AtomicWriteFunc(g.cfg.ConfigPath, data, 0644, g.validate); err != nil {
		return false, err
	}

	g.mu.Lock()
	g.hash = h
	g.mu.Unlock()
	return true, nil
}

// logDenyAllEndpoints warns for each endpoint that ended up with the deny-all
// sentinel — required scopes that no role covers (access denied for everyone).
func logDenyAllEndpoints(endpoints []interface{}) {
	for _, ep := range endpoints {
		epMap, ok := ep.(map[string]interface{})
		if !ok {
			continue
		}
		extra, ok := epMap["extra_config"].(map[string]interface{})
		if !ok {
			continue
		}
		validator, ok := extra["auth/validator"].(map[string]interface{})
		if !ok {
			continue
		}
		roles, ok := validator["roles"].([]string)
		if !ok {
			continue
		}
		for _, r := range roles {
			if r == denyAllRole {
				log.Printf("gateway: endpoint %s %s has required scopes but no role covers them — access denied for all",
					epMap["method"], epMap["endpoint"])
				break
			}
		}
	}
}

// Render fetches specs and returns the complete krakend.json bytes without
// writing, validating, or touching the change-detection hash. Used by --dry-run
// to preview exactly what would be generated.
func (g *Generator) Render(services []ServiceInfo, scopeRoles map[string][]string) ([]byte, error) {
	data, _, _, err := g.build(services, scopeRoles)
	return data, err
}

// RenderWithDigests is Render plus a per-service digest (info.version + contract
// hash) used by the artifact-mode version gate. Computed from the same fetched
// specs, so it costs no extra requests.
func (g *Generator) RenderWithDigests(services []ServiceInfo, scopeRoles map[string][]string) ([]byte, map[string]versiongate.Digest, error) {
	data, _, digests, err := g.build(services, scopeRoles)
	return data, digests, err
}

// build fetches every service spec, derives the KrakenD endpoints, and marshals
// the complete config. It returns the bytes, the endpoint list (for change-time
// logging in Generate), and a per-service digest (for the version gate).
func (g *Generator) build(services []ServiceInfo, scopeRoles map[string][]string) ([]byte, []interface{}, map[string]versiongate.Digest, error) {
	var endpoints []interface{}
	digests := make(map[string]versiongate.Digest)
	for _, svc := range services {
		raw, err := g.fetchSpec(svc)
		if err != nil {
			log.Printf("gateway: skip %s: %v", svc.Name, err)
			continue
		}
		endpoints = append(endpoints, g.endpointsFromSpec(svc, raw, scopeRoles)...)
		digests[svc.Name] = versiongate.Digest{
			Version:      specVersion(raw),
			ContractHash: versiongate.ContractHash(raw),
		}
	}

	data, err := g.marshalConfig(endpoints)
	if err != nil {
		return nil, nil, nil, err
	}
	return data, endpoints, digests, nil
}

// --- spec fetching (mirrors openapi package — kept separate to avoid cross-package dependency) ---

func (g *Generator) fetchSpec(svc ServiceInfo) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s:%d/%s", svc.Name, svc.Port, svc.OpenAPIRoute)
	resp, err := g.http.Get(url) //nolint:gosec
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

func (g *Generator) endpointsFromSpec(svc ServiceInfo, raw map[string]interface{}, scopeRoles map[string][]string) []interface{} {
	version := specVersion(raw)
	prefix := fmt.Sprintf("/%s/%s/%s", g.cfg.APIRoute, svc.Name, version)
	host := fmt.Sprintf("http://%s:%d", svc.Name, svc.Port)

	var eps []interface{}
	rawPaths, _ := raw["paths"].(map[string]interface{})
	paths := make([]string, 0, len(rawPaths))
	for p := range rawPaths {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	for _, rawPath := range paths {
		pathItem := rawPaths[rawPath]
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
			scopes := extractScopes(op)
			matcher := scopesMatcher(op)
			roles := resolveRoles(scopes, matcher, scopeRoles)
			ep := g.buildEndpoint(
				strings.ToUpper(method),
				prefix+rawPath,
				rawPath,
				host,
				protected,
				roles,
				responseIsCollection(op),
			)
			eps = append(eps, ep)
		}
	}
	return eps
}

func (g *Generator) buildEndpoint(method, gatewayPath, backendPath, host string, protected bool, roles []string, collection bool) map[string]interface{} {
	backend := map[string]interface{}{
		"url_pattern": backendPath,
		"method":      method,
		"host":        []string{host},
		"encoding":    "json",
		"extra_config": map[string]interface{}{
			"backend/http": map[string]interface{}{
				"return_error_details": "backend",
				"return_error_code":    true,
			},
		},
	}
	if collection {
		backend["is_collection"] = true
	}

	outputEncoding := "json"
	if collection {
		outputEncoding = "json-collection"
	}

	ep := map[string]interface{}{
		"endpoint":        gatewayPath,
		"method":          method,
		"output_encoding": outputEncoding,
		"backend":         []map[string]interface{}{backend},
	}

	if !protected {
		return ep
	}

	jwkURL := fmt.Sprintf("%s://%s.%s/oidc/jwks",
		g.cfg.HTTPProtocol, g.cfg.OIDCSubdomain, g.cfg.BaseDomain)

	validator := map[string]interface{}{
		"alg":                  "ES384",
		"jwk_url":              jwkURL,
		"disable_jwk_security": g.cfg.HTTPProtocol == "http",
		"audience":             []string{g.cfg.LogtoResourceID},
		"propagate_claims": [][]string{
			{"sub", "x-user-id"},
			{"roles", "x-user-roles"},
			{"organization_id", "x-organization-id"},
		},
	}
	if len(roles) > 0 {
		validator["roles"] = roles
		validator["roles_key"] = "roles"
		validator["roles_key_is_nested"] = false
	}

	ep["extra_config"] = map[string]interface{}{
		"auth/validator": validator,
	}
	headers := []string{"Authorization"}
	if method == "POST" || method == "PUT" || method == "PATCH" {
		headers = append(headers, "Content-Type")
	}
	ep["input_headers"] = headers
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

	// Inject CORS into the service-level extra_config so the admin SPA (a
	// different origin than the API) can call the gateway from the browser.
	// KrakenD's security/cors handles the preflight (OPTIONS) automatically.
	if len(g.cfg.CORSAllowOrigins) > 0 {
		extra, ok := config["extra_config"].(map[string]interface{})
		if !ok {
			extra = map[string]interface{}{}
			config["extra_config"] = extra
		}
		extra["security/cors"] = map[string]interface{}{
			"allow_origins":     g.cfg.CORSAllowOrigins,
			"allow_methods":     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			"allow_headers":     []string{"Origin", "Authorization", "Content-Type"},
			"expose_headers":    []string{"Content-Length"},
			"allow_credentials": false,
			"max_age":           "12h",
		}
	}

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

// denyAllRole is a sentinel role that no real user holds. It is assigned to an
// endpoint whenever required scopes cannot be satisfied — either because the
// scope→role mapping is unavailable (Logto init not yet complete) or because no
// role covers the required scopes. KrakenD then rejects every request to that
// endpoint: fail closed, never fail open.
const denyAllRole = "__no_role_configured__"

// resolveRoles maps required scopes to the roles that satisfy them.
// matcher "any": union of roles that have at least one required scope.
// matcher "all": intersection of roles that have all required scopes.
// Returns nil when no scopes are required (the endpoint carries no role
// restriction). When scopes ARE required but cannot be resolved, returns the
// deny-all sentinel so the endpoint fails closed rather than admitting any valid JWT.
func resolveRoles(scopes []string, matcher string, scopeRoles map[string][]string) []string {
	if len(scopes) == 0 {
		return nil
	}
	// Scopes are required → the endpoint must be role-restricted. If the mapping
	// is unavailable, deny all rather than letting every authenticated caller through.
	if scopeRoles == nil {
		return []string{denyAllRole}
	}

	var result []string
	if matcher == "all" {
		// Start with roles that have the first scope, then intersect.
		first := true
		for _, s := range scopes {
			roleSet := toSet(scopeRoles[s])
			if first {
				result = keys(roleSet)
				first = false
				continue
			}
			var intersect []string
			for _, r := range result {
				if roleSet[r] {
					intersect = append(intersect, r)
				}
			}
			result = intersect
		}
	} else {
		// "any": union of all roles that have at least one scope.
		seen := map[string]bool{}
		for _, s := range scopes {
			for _, r := range scopeRoles[s] {
				if !seen[r] {
					seen[r] = true
					result = append(result, r)
				}
			}
		}
	}
	sort.Strings(result)

	// Scopes are required but no role covers them → deny all.
	if len(result) == 0 {
		return []string{denyAllRole}
	}
	return result
}

func toSet(slice []string) map[string]bool {
	m := make(map[string]bool, len(slice))
	for _, s := range slice {
		m[s] = true
	}
	return m
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// responseIsCollection returns true if the operation's 200/201 response schema is an array.
func responseIsCollection(op map[string]interface{}) bool {
	responses, ok := op["responses"].(map[string]interface{})
	if !ok {
		return false
	}
	for _, code := range []string{"200", "201"} {
		resp, ok := responses[code].(map[string]interface{})
		if !ok {
			continue
		}
		content, ok := resp["content"].(map[string]interface{})
		if !ok {
			continue
		}
		for _, mediaType := range content {
			mt, ok := mediaType.(map[string]interface{})
			if !ok {
				continue
			}
			schema, ok := mt["schema"].(map[string]interface{})
			if !ok {
				continue
			}
			if schema["type"] == "array" {
				return true
			}
		}
	}
	return false
}

var httpMethods = []string{
	"get", "post", "put", "patch", "delete", "head", "options", "trace",
}
