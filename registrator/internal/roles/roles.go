// Package roles supplies the scope→roles mapping the gateway uses to turn an
// endpoint's required x-infra-scopes into the set of roles KrakenD will accept.
//
// Under Logto this mapping was fetched from the IdP (roles carried scopes). Zitadel
// project roles are plain keys with no attached scopes, so "which roles satisfy
// which scope" is gateway authorization policy and lives here — independent of the
// IdP. The IdP only defines role *membership*; the role *keys* must match those the
// bootstrap creates (see scripts/zitadel/zitadel.config.yaml): admin, org_owner,
// org_user.
package roles

import (
	"encoding/json"
	"fmt"
	"os"
)

// defaultRoleScopes maps each role to the scopes it grants. Mirrors the role/scope
// model previously declared in logto.config.yaml. Override with the env var
// INFRA_GATEWAY_ROLE_SCOPES (JSON: {"role":["scope",...]}).
var defaultRoleScopes = map[string][]string{
	"admin": {
		"read:items", "write:items", "delete:items",
		"read:organizations", "write:organizations", "delete:organizations",
	},
	"org_owner": {"read:items", "write:items", "delete:items"},
	"org_user":  {"read:items", "write:items"},
}

// Provider holds the inverted scope→roles mapping.
type Provider struct {
	scopeRoles map[string][]string
}

// New builds the provider from INFRA_GATEWAY_ROLE_SCOPES if set, else the built-in
// default. Returns an error if the override is set but malformed (callers treat a
// nil mapping as fail-closed → scoped endpoints deny all).
func New() (*Provider, error) {
	roleScopes := defaultRoleScopes
	if raw := os.Getenv("INFRA_GATEWAY_ROLE_SCOPES"); raw != "" {
		parsed := map[string][]string{}
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return nil, fmt.Errorf("roles: invalid INFRA_GATEWAY_ROLE_SCOPES: %w", err)
		}
		roleScopes = parsed
	}

	scopeRoles := map[string][]string{}
	for role, scopes := range roleScopes {
		for _, s := range scopes {
			scopeRoles[s] = append(scopeRoles[s], role)
		}
	}
	return &Provider{scopeRoles: scopeRoles}, nil
}

// ScopeRoles returns the scope→roles mapping. The error return mirrors the former
// Logto client signature so the reload path is unchanged; it is always nil here
// because the mapping is static once built.
func (p *Provider) ScopeRoles() (map[string][]string, error) {
	return p.scopeRoles, nil
}
