package logto

import (
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

const credentialsPath = "/run/infra/registrator-m2m.json"

type credentials struct {
	ClientID      string `json:"clientId"`
	ClientSecret  string `json:"clientSecret"`
	TokenEndpoint string `json:"tokenEndpoint"`
	APIEndpoint   string `json:"apiEndpoint"`
}

type role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type scope struct {
	Name string `json:"name"`
}

// Client fetches the scope→roles mapping from the Logto Management API.
type Client struct {
	mu       sync.Mutex
	token    string
	tokenExp time.Time
	creds    *credentials
}

func New() *Client {
	return &Client{}
}

// ScopeRoles returns a map of scope name → role names that have that scope.
// Merges both user roles and organization roles (resource scopes).
// Returns nil if credentials are not yet available (init hasn't finished).
func (c *Client) ScopeRoles() (map[string][]string, error) {
	if err := c.loadCreds(); err != nil {
		return nil, err
	}
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)

	// User roles and their resource scopes.
	roles, err := c.fetchRoles(token)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		scopes, err := c.fetchRoleScopes(token, r.ID)
		if err != nil {
			return nil, fmt.Errorf("fetch scopes for role %s: %w", r.Name, err)
		}
		for _, s := range scopes {
			result[s.Name] = append(result[s.Name], r.Name)
		}
	}

	// Organization roles and their resource scopes — merged into the same flat map.
	orgRoles, err := c.fetchOrgRoles(token)
	if err != nil {
		return nil, fmt.Errorf("fetch org roles: %w", err)
	}
	for _, r := range orgRoles {
		scopes, err := c.fetchOrgRoleResourceScopes(token, r.ID)
		if err != nil {
			return nil, fmt.Errorf("fetch resource scopes for org role %s: %w", r.Name, err)
		}
		for _, s := range scopes {
			result[s.Name] = append(result[s.Name], r.Name)
		}
	}

	return result, nil
}

func (c *Client) loadCreds() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.creds != nil {
		return nil
	}
	data, err := os.ReadFile(credentialsPath)
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

func (c *Client) getToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}

	body := url.Values{
		"grant_type": {"client_credentials"},
		"resource":   {"https://default.logto.app/api"},
		"scope":      {"all"},
	}
	req, _ := http.NewRequest("POST", c.creds.TokenEndpoint+"/oidc/token", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(c.creds.ClientID, c.creds.ClientSecret)

	resp, err := http.DefaultClient.Do(req)
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

func (c *Client) fetchRoles(token string) ([]role, error) {
	var all []role
	page := 1
	for {
		var batch []role
		if err := c.get(token, fmt.Sprintf("/api/roles?type=User&page=%d&page_size=50", page), &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 50 {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) fetchRoleScopes(token, roleID string) ([]scope, error) {
	var scopes []scope
	err := c.get(token, fmt.Sprintf("/api/roles/%s/scopes?page_size=50", roleID), &scopes)
	return scopes, err
}

func (c *Client) fetchOrgRoles(token string) ([]role, error) {
	var all []role
	page := 1
	for {
		var batch []role
		if err := c.get(token, fmt.Sprintf("/api/organization-roles?page=%d&page_size=50", page), &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 50 {
			break
		}
		page++
	}
	return all, nil
}

func (c *Client) fetchOrgRoleResourceScopes(token, roleID string) ([]scope, error) {
	var scopes []scope
	err := c.get(token, fmt.Sprintf("/api/organization-roles/%s/resource-scopes?page_size=50", roleID), &scopes)
	return scopes, err
}

func (c *Client) get(token, path string, out interface{}) error {
	req, _ := http.NewRequest("GET", c.creds.APIEndpoint+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("logto: GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("logto: GET %s → %d: %s", path, resp.StatusCode, body)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
