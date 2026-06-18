// Package logto is a write-capable Logto Management API client for service-core.
// service-core is the source of truth for organizations; this client provisions
// them in Logto (the identity provider) as a downstream follower. Identity-
// provider specifics are confined here — the rest of the service speaks in terms
// of a provider-neutral external_id.
package logto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// httpTimeout bounds every outbound Logto request. Outbox delivery runs under a
// per-event DB row lock, so a hung call must not stall indefinitely.
const httpTimeout = 10 * time.Second

type credentials struct {
	ClientID      string `json:"clientId"`
	ClientSecret  string `json:"clientSecret"`
	TokenEndpoint string `json:"tokenEndpoint"`
	APIEndpoint   string `json:"apiEndpoint"`
}

// Client talks to the Logto Management API using M2M client-credentials.
type Client struct {
	credsPath     string
	tenantAppPath string

	mu          sync.Mutex
	token       string
	tokenExp    time.Time
	creds       *credentials
	tenantAppID string
	http        *http.Client
}

// New returns a client that reads its M2M credentials from credsPath and the
// shared Tenant application id from tenantAppPath on first use (both written by
// logto-init, which may not be ready at startup).
func New(credsPath, tenantAppPath string) *Client {
	return &Client{
		credsPath:     credsPath,
		tenantAppPath: tenantAppPath,
		http:          &http.Client{Timeout: httpTimeout},
	}
}

type organization struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	CustomData map[string]any `json:"customData"`
}

// Create provisions an organization in Logto, stamping the core ID on customData
// so it can be reconciled later, and returns the Logto organization ID.
func (c *Client) Create(ctx context.Context, name, description, coreID string) (string, error) {
	body := map[string]any{
		"name":        name,
		"description": description,
		"customData":  map[string]any{"coreOrgId": coreID},
	}
	var org organization
	if err := c.do(ctx, http.MethodPost, "/api/organizations", body, &org); err != nil {
		return "", err
	}
	if org.ID == "" {
		return "", fmt.Errorf("logto: create organization returned empty id")
	}
	return org.ID, nil
}

// FindByCoreID locates an organization previously stamped with the given core ID.
// Used to recover idempotently from a crash between Create and the local commit.
func (c *Client) FindByCoreID(ctx context.Context, coreID string) (string, bool, error) {
	page := 1
	for {
		var batch []organization
		path := fmt.Sprintf("/api/organizations?page=%d&page_size=50", page)
		if err := c.do(ctx, http.MethodGet, path, nil, &batch); err != nil {
			return "", false, err
		}
		for _, o := range batch {
			if id, _ := o.CustomData["coreOrgId"].(string); id == coreID {
				return o.ID, true, nil
			}
		}
		if len(batch) < 50 {
			return "", false, nil
		}
		page++
	}
}

// EnsureTenantRedirectURI registers the tenant frontend's OIDC redirect URIs for
// the given origin on the shared Tenant application. Idempotent: it appends
// "<origin>/callback" to redirectUris and "<origin>" to postLogoutRedirectUris
// only if missing, preserving any other oidcClientMetadata fields.
func (c *Client) EnsureTenantRedirectURI(ctx context.Context, origin string) error {
	appID, err := c.loadTenantAppID()
	if err != nil {
		return err
	}

	var app struct {
		OidcClientMetadata map[string]any `json:"oidcClientMetadata"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/applications/"+appID, nil, &app); err != nil {
		return err
	}

	meta := app.OidcClientMetadata
	if meta == nil {
		meta = map[string]any{}
	}
	redirects := toStringSlice(meta["redirectUris"])
	postLogouts := toStringSlice(meta["postLogoutRedirectUris"])

	callback := origin + "/callback"
	changed := false
	if !contains(redirects, callback) {
		redirects = append(redirects, callback)
		changed = true
	}
	if !contains(postLogouts, origin) {
		postLogouts = append(postLogouts, origin)
		changed = true
	}
	if !changed {
		return nil
	}

	meta["redirectUris"] = redirects
	meta["postLogoutRedirectUris"] = postLogouts
	body := map[string]any{"oidcClientMetadata": meta}
	return c.do(ctx, http.MethodPatch, "/api/applications/"+appID, body, nil)
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func toStringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, e := range raw {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (c *Client) loadTenantAppID() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tenantAppID != "" {
		return c.tenantAppID, nil
	}
	data, err := os.ReadFile(c.tenantAppPath)
	if err != nil {
		return "", fmt.Errorf("logto: tenant app config not ready (%w)", err)
	}
	var t struct {
		AppID string `json:"appId"`
	}
	if err := json.Unmarshal(data, &t); err != nil {
		return "", fmt.Errorf("logto: invalid tenant app config: %w", err)
	}
	if t.AppID == "" {
		return "", fmt.Errorf("logto: tenant app id empty in %s", c.tenantAppPath)
	}
	c.tenantAppID = t.AppID
	return t.AppID, nil
}

func (c *Client) loadCreds() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.creds != nil {
		return nil
	}
	data, err := os.ReadFile(c.credsPath)
	if err != nil {
		return fmt.Errorf("logto: credentials not ready (%w)", err)
	}
	var creds credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("logto: invalid credentials file: %w", err)
	}
	c.creds = &creds
	return nil
}

func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}

	form := url.Values{
		"grant_type": {"client_credentials"},
		"resource":   {"https://default.logto.app/api"},
		"scope":      {"all"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.creds.TokenEndpoint+"/oidc/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.creds.ClientID, c.creds.ClientSecret)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("logto: token request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("logto: decode token response: %w", err)
	}
	if result.Error != "" || result.AccessToken == "" {
		return "", fmt.Errorf("logto: token error: %s", result.Error)
	}

	c.token = result.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(result.ExpiresIn-30) * time.Second)
	return c.token, nil
}

// do performs an authenticated Management API request. body and out are optional.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	if err := c.loadCreds(); err != nil {
		return err
	}
	token, err := c.getToken(ctx)
	if err != nil {
		return err
	}

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.creds.APIEndpoint+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("logto: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logto: %s %s → %d: %s", method, path, resp.StatusCode, b)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
