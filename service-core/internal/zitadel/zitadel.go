// Package zitadel is a write-capable Zitadel Management API client for service-core.
// service-core is the source of truth for organizations; this client provisions
// them in Zitadel (the identity provider) as a downstream follower. Identity-
// provider specifics are confined here — the rest of the service speaks in terms
// of a provider-neutral external_id (the Zitadel organization id).
//
// Auth is a long-lived Personal Access Token (PAT) issued to a machine user by
// zitadel-init and written to credsPath; no client-credentials token exchange is
// needed (unlike the former Logto client).
package zitadel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	orgdom "infra/service-core/internal/organization"
)

// httpTimeout bounds every outbound Zitadel request. Outbox delivery runs under a
// per-event DB row lock, so a hung call must not stall indefinitely.
const httpTimeout = 10 * time.Second

type credentials struct {
	Token       string `json:"token"`
	APIEndpoint string `json:"apiEndpoint"`
}

type tenantApp struct {
	OrgID     string `json:"orgId"`
	ProjectID string `json:"projectId"`
	AppID     string `json:"appId"`
}

// Client talks to the Zitadel Management API with a machine-user PAT.
type Client struct {
	credsPath     string
	tenantAppPath string

	mu    sync.Mutex
	creds *credentials
	app   *tenantApp
	http  *http.Client
}

// New returns a client that reads its PAT from credsPath and the shared Tenant
// application descriptor from tenantAppPath on first use (both written by
// zitadel-init, which may not be ready at startup).
func New(credsPath, tenantAppPath string) *Client {
	return &Client{
		credsPath:     credsPath,
		tenantAppPath: tenantAppPath,
		http:          &http.Client{Timeout: httpTimeout},
	}
}

// Create provisions an organization in Zitadel and returns its id. Zitadel
// organizations carry no description field, so description is ignored. Zitadel
// enforces unique organization names, which we use as the idempotency key: a 409 on
// (re)create means a prior attempt already made this org, so we adopt it by name.
// (Zitadel v4 removed the v1 org-metadata endpoints we would otherwise have used to
// stamp/look up the core id, so the unique name is our reconciliation handle.)
func (c *Client) Create(ctx context.Context, name, _, _ string) (string, error) {
	var res struct {
		ID string `json:"id"`
	}
	err := c.do(ctx, http.MethodPost, "/management/v1/orgs", "", map[string]any{"name": name}, &res)
	if err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) && apiErr.status == http.StatusConflict {
			if id, found, ferr := c.findOrgByName(ctx, name); ferr != nil {
				return "", ferr
			} else if found {
				return id, nil
			}
		}
		return "", err
	}
	if res.ID == "" {
		return "", fmt.Errorf("zitadel: create org returned empty id")
	}
	return res.ID, nil
}

// FindByCoreID is the outbox's crash-recovery hook (called before a retry's Create).
// Zitadel v4 removed the v1 org-metadata endpoints we previously used to stamp/look
// up the core id, so there is no metadata to query here; recovery instead happens in
// Create, which adopts an existing org by its unique name on a 409. Reporting
// "not found" lets the caller proceed to Create, which performs that recovery.
func (c *Client) FindByCoreID(_ context.Context, _ string) (string, bool, error) {
	return "", false, nil
}

// findOrgByName looks up an organization by its exact (unique) name.
func (c *Client) findOrgByName(ctx context.Context, name string) (string, bool, error) {
	var res struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	q := map[string]any{
		"queries": []map[string]any{
			{"nameQuery": map[string]any{"name": name, "method": "TEXT_QUERY_METHOD_EQUALS"}},
		},
	}
	if err := c.do(ctx, http.MethodPost, "/admin/v1/orgs/_search", "", q, &res); err != nil {
		return "", false, err
	}
	if len(res.Result) == 0 {
		return "", false, nil
	}
	return res.Result[0].ID, true, nil
}

// EnsureRoleAccess grants the shared Infra API project to the given organization
// (a Zitadel "project grant"), so its members can be assigned org_owner/org_user.
// Without a project grant, a user grant in a tenant org asserts no roles in the
// token. Idempotent: skips if a grant for the org already exists. Tenant orgs are
// granted org_owner/org_user only — never the platform "admin" role (that would let
// a tenant assign platform-wide admin to its own users).
func (c *Client) EnsureRoleAccess(ctx context.Context, externalID string) error {
	app, err := c.loadTenantApp()
	if err != nil {
		return err
	}
	base := fmt.Sprintf("/management/v1/projects/%s/grants", app.ProjectID)

	var res struct {
		Result []struct {
			GrantedOrgID string `json:"grantedOrgId"`
		} `json:"result"`
	}
	if err := c.do(ctx, http.MethodPost, base+"/_search", app.OrgID, map[string]any{}, &res); err != nil {
		return err
	}
	for _, g := range res.Result {
		if g.GrantedOrgID == externalID {
			return nil
		}
	}

	body := map[string]any{"grantedOrgId": externalID, "roleKeys": []string{"org_owner", "org_user"}}
	return c.do(ctx, http.MethodPost, base, app.OrgID, body, nil)
}

// EnsureTenantRedirectURI registers the tenant frontend's OIDC redirect URIs for
// the given origin on the shared Tenant application. Idempotent: it appends
// "<origin>/callback" to redirectUris and "<origin>" to postLogoutRedirectUris and
// additionalOrigins only if missing, preserving any other config. Zitadel rejects
// a no-op update (400 "No changes"), so it PATCHes only when something changed.
func (c *Client) EnsureTenantRedirectURI(ctx context.Context, origin string) error {
	app, err := c.loadTenantApp()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/management/v1/projects/%s/apps/%s", app.ProjectID, app.AppID)

	var cur struct {
		App struct {
			OIDCConfig struct {
				RedirectURIs           []string `json:"redirectUris"`
				PostLogoutRedirectURIs []string `json:"postLogoutRedirectUris"`
				AdditionalOrigins      []string `json:"additionalOrigins"`
			} `json:"oidcConfig"`
		} `json:"app"`
	}
	if err := c.do(ctx, http.MethodGet, path, app.OrgID, nil, &cur); err != nil {
		return err
	}
	cfg := cur.App.OIDCConfig

	callback := origin + "/callback"
	redirects, c1 := appendMissing(cfg.RedirectURIs, callback)
	postLogouts, c2 := appendMissing(cfg.PostLogoutRedirectURIs, origin)
	origins, c3 := appendMissing(cfg.AdditionalOrigins, origin)
	if !c1 && !c2 && !c3 {
		return nil
	}

	body := map[string]any{
		"redirectUris":           redirects,
		"postLogoutRedirectUris": postLogouts,
		"additionalOrigins":      origins,
		"responseTypes":          []string{"OIDC_RESPONSE_TYPE_CODE"},
		"grantTypes":             []string{"OIDC_GRANT_TYPE_AUTHORIZATION_CODE"},
		"appType":                "OIDC_APP_TYPE_USER_AGENT",
		"authMethodType":         "OIDC_AUTH_METHOD_TYPE_NONE",
		"accessTokenType":        "OIDC_TOKEN_TYPE_JWT",
	}
	return c.do(ctx, http.MethodPut, path+"/oidc_config", app.OrgID, body, nil)
}

// grant is a user's project grant as returned by the user-grant search: it carries
// the member's profile inline alongside their role keys, so one call yields members
// and their org-scoped roles.
type grant struct {
	UserID      string   `json:"userId"`
	DisplayName string   `json:"displayName"`
	Email       string   `json:"email"`
	UserName    string   `json:"userName"`
	AvatarURL   string   `json:"avatarUrl"`
	RoleKeys    []string `json:"roleKeys"`
}

// ListMembers returns an organization's members and their org-scoped roles from
// Zitadel. orgID is the Zitadel organization id (the org's external_id). Membership
// is modeled as user grants on the shared Infra API project, so a single
// user-grant search scoped to the org yields members + their role keys. The page is
// 1-based. Returns provider-neutral organization.Member values.
func (c *Client) ListMembers(ctx context.Context, orgID string, page, pageSize int, q string) ([]orgdom.Member, int, error) {
	app, err := c.loadTenantApp()
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	queries := []map[string]any{
		{"projectIdQuery": map[string]any{"projectId": app.ProjectID}},
	}
	if q != "" {
		queries = append(queries, map[string]any{"displayNameQuery": map[string]any{"displayName": q, "method": "TEXT_QUERY_METHOD_CONTAINS_IGNORE_CASE"}})
	}
	reqBody := map[string]any{
		"query":   map[string]any{"offset": strconv.Itoa(offset), "limit": pageSize, "asc": true},
		"queries": queries,
	}

	var res struct {
		Details struct {
			TotalResult string `json:"totalResult"`
		} `json:"details"`
		Result []grant `json:"result"`
	}
	if err := c.do(ctx, http.MethodPost, "/management/v1/users/grants/_search", orgID, reqBody, &res); err != nil {
		return nil, 0, err
	}

	members := make([]orgdom.Member, len(res.Result))
	for i, g := range res.Result {
		name := g.DisplayName
		if name == "" {
			name = g.UserName
		}
		roles := make([]orgdom.MemberRole, len(g.RoleKeys))
		for j, k := range g.RoleKeys {
			roles[j] = orgdom.MemberRole{ID: k, Name: k}
		}
		members[i] = orgdom.Member{
			ID:     g.UserID,
			Name:   name,
			Email:  g.Email,
			Avatar: g.AvatarURL,
			Roles:  roles,
		}
	}
	total, _ := strconv.Atoi(res.Details.TotalResult)
	return members, total, nil
}

func appendMissing(s []string, v string) ([]string, bool) {
	for _, x := range s {
		if x == v {
			return s, false
		}
	}
	return append(s, v), true
}

func (c *Client) loadCreds() (*credentials, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.creds != nil {
		return c.creds, nil
	}
	data, err := os.ReadFile(c.credsPath)
	if err != nil {
		return nil, fmt.Errorf("zitadel: credentials not ready (%w)", err)
	}
	var creds credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("zitadel: invalid credentials file: %w", err)
	}
	if creds.Token == "" || creds.APIEndpoint == "" {
		return nil, fmt.Errorf("zitadel: credentials file missing token or apiEndpoint")
	}
	c.creds = &creds
	return c.creds, nil
}

func (c *Client) loadTenantApp() (*tenantApp, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.app != nil {
		return c.app, nil
	}
	data, err := os.ReadFile(c.tenantAppPath)
	if err != nil {
		return nil, fmt.Errorf("zitadel: tenant app config not ready (%w)", err)
	}
	var app tenantApp
	if err := json.Unmarshal(data, &app); err != nil {
		return nil, fmt.Errorf("zitadel: invalid tenant app config: %w", err)
	}
	if app.OrgID == "" || app.ProjectID == "" || app.AppID == "" {
		return nil, fmt.Errorf("zitadel: tenant app config incomplete in %s", c.tenantAppPath)
	}
	c.app = &app
	return c.app, nil
}

// apiError carries the HTTP status so callers can distinguish e.g. 404.
type apiError struct {
	status int
	method string
	path   string
	body   string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("zitadel: %s %s -> %d: %s", e.method, e.path, e.status, e.body)
}

// do performs an authenticated Management/Admin API request. orgID, when non-empty,
// scopes the call to that organization via the x-zitadel-orgid header. body and out
// are optional.
func (c *Client) do(ctx context.Context, method, path, orgID string, body, out any) error {
	creds, err := c.loadCreds()
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

	req, err := http.NewRequestWithContext(ctx, method, creds.APIEndpoint+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+creds.Token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("zitadel: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return &apiError{status: resp.StatusCode, method: method, path: path, body: string(b)}
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return err
		}
	}
	return nil
}
