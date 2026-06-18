package organization

import (
	"context"
	"testing"

	"infra/service-core/internal/outbox"
)

func TestToResponseSyncedFlag(t *testing.T) {
	ext := "logto-org-123"
	if r := toResponse(Organization{ExternalID: &ext}); !r.Synced || r.ExternalID == nil {
		t.Errorf("org with external_id should be synced: %+v", r)
	}
	if r := toResponse(Organization{}); r.Synced {
		t.Error("org without external_id must not be synced")
	}
	empty := ""
	if r := toResponse(Organization{ExternalID: &empty}); r.Synced {
		t.Error("org with empty external_id must not be synced")
	}
}

func TestOutboxHandlerRejectsUnknownEvent(t *testing.T) {
	// Unknown event types are rejected before any DB or provider access, so a nil
	// provider and tx are never dereferenced.
	h := NewOutboxHandler(nil)
	if err := h(context.Background(), nil, outbox.Event{EventType: "nonsense"}); err == nil {
		t.Fatal("expected error for unknown event type")
	}
}
