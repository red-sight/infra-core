package gateway

import (
	"reflect"
	"testing"
)

func TestResolveRoles(t *testing.T) {
	scopeRoles := map[string][]string{
		"read:orgs":    {"admin", "viewer"},
		"write:orgs":   {"admin"},
		"read:users":   {"admin", "support"},
		"read:reports": {"viewer"}, // no admin — used to force an empty intersection
	}

	tests := []struct {
		name       string
		scopes     []string
		matcher    string
		scopeRoles map[string][]string
		want       []string
	}{
		{
			name:       "no scopes means no role restriction",
			scopes:     nil,
			matcher:    "any",
			scopeRoles: scopeRoles,
			want:       nil,
		},
		{
			name:       "scopes required but mapping unavailable fails closed",
			scopes:     []string{"read:orgs"},
			matcher:    "any",
			scopeRoles: nil, // Logto not ready
			want:       []string{denyAllRole},
		},
		{
			name:       "any is the union of roles holding at least one scope",
			scopes:     []string{"write:orgs", "read:users"},
			matcher:    "any",
			scopeRoles: scopeRoles,
			want:       []string{"admin", "support"},
		},
		{
			name:       "all is the intersection of roles holding every scope",
			scopes:     []string{"read:orgs", "write:orgs"},
			matcher:    "all",
			scopeRoles: scopeRoles,
			want:       []string{"admin"},
		},
		{
			name:       "all with empty intersection fails closed",
			scopes:     []string{"write:orgs", "read:reports"},
			matcher:    "all",
			scopeRoles: scopeRoles, // write:orgs→admin, read:reports→viewer — no role has both
			want:       []string{denyAllRole},
		},
		{
			name:       "any with no covering role fails closed",
			scopes:     []string{"delete:everything"},
			matcher:    "any",
			scopeRoles: scopeRoles,
			want:       []string{denyAllRole},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveRoles(tt.scopes, tt.matcher, tt.scopeRoles)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("resolveRoles(%v, %q) = %v, want %v", tt.scopes, tt.matcher, got, tt.want)
			}
		})
	}
}
