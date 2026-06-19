package organization

import (
	"context"
	"testing"

	"infra/service-core/internal/outbox"
)

func TestToResponseSyncedFlag(t *testing.T) {
	h := &handler{}
	ext := "logto-org-123"
	if r := h.toResponse(Organization{ExternalID: &ext}); !r.Synced || r.ExternalID == nil {
		t.Errorf("org with external_id should be synced: %+v", r)
	}
	if r := h.toResponse(Organization{}); r.Synced {
		t.Error("org without external_id must not be synced")
	}
	empty := ""
	if r := h.toResponse(Organization{ExternalID: &empty}); r.Synced {
		t.Error("org with empty external_id must not be synced")
	}
}

func TestValidateSlug(t *testing.T) {
	valid := []string{"acme", "ac", "a-b-c", "org-123", "x1"}
	for _, s := range valid {
		if err := validateSlug(s); err != nil {
			t.Errorf("validateSlug(%q) = %v, want nil", s, err)
		}
	}
	invalid := []string{
		"a",          // too short
		"-acme",      // leading hyphen
		"acme-",      // trailing hyphen
		"Acme",       // uppercase
		"ac me",      // space
		"admin",      // reserved
		"api",        // reserved
		"auth-admin", // reserved
	}
	for _, s := range invalid {
		if err := validateSlug(s); err == nil {
			t.Errorf("validateSlug(%q) = nil, want error", s)
		}
	}
}

func TestOutboxHandlerRejectsUnknownEvent(t *testing.T) {
	// Unknown event types are rejected before any DB or provider access, so a nil
	// provider and tx are never dereferenced.
	h := NewOutboxHandler(nil, HandlerConfig{})
	if err := h(context.Background(), nil, outbox.Event{EventType: "nonsense"}); err == nil {
		t.Fatal("expected error for unknown event type")
	}
}
