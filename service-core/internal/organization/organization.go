package organization

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/red-sight/infra-core/query"

	"infra/service-core/internal/outbox"
)

type Organization struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	ExternalID  *string   `gorm:"uniqueIndex"           json:"external_id"`
	Name        string    `gorm:"not null"              json:"name"`
	Description string    `gorm:"not null;default:''"   json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrgResponse struct {
	ID          string    `json:"id"          doc:"Internal organization ID"`
	ExternalID  *string   `json:"external_id" doc:"Identity-provider organization ID (Logto). Null until synced."`
	Synced      bool      `json:"synced"      doc:"Whether the organization has been provisioned in the identity provider."`
	Name        string    `json:"name"        doc:"Organization name"`
	Description string    `json:"description" doc:"Organization description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toResponse(o Organization) OrgResponse {
	return OrgResponse{
		ID:          o.ID,
		ExternalID:  o.ExternalID,
		Synced:      o.ExternalID != nil && *o.ExternalID != "",
		Name:        o.Name,
		Description: o.Description,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

func RegisterRoutes(api huma.API, db *gorm.DB) {
	h := &handler{db: db}

	huma.Register(api, huma.Operation{
		OperationID: "list-organizations",
		Method:      http.MethodGet,
		Path:        "/admin/organizations",
		Summary:     "List organizations",
		Description: "Returns a paginated, searchable list of organizations. Sortable by: name, created_at, updated_at.",
		Tags:        []string{"Admin"},
		Extensions: map[string]any{
			"x-infra-protected": true,
			"x-infra-scopes":    []string{"read:organizations"},
		},
	}, h.list)

	huma.Register(api, huma.Operation{
		OperationID:   "create-organization",
		Method:        http.MethodPost,
		Path:          "/admin/organizations",
		DefaultStatus: http.StatusCreated,
		Summary:       "Create organization",
		Description:   "Creates an organization. Core is the source of truth; the organization is provisioned in the identity provider asynchronously, so the response returns synced=false until the outbox worker completes.",
		Tags:          []string{"Admin"},
		Extensions: map[string]any{
			"x-infra-protected": true,
			"x-infra-scopes":    []string{"write:organizations"},
		},
	}, h.create)
}

type handler struct{ db *gorm.DB }

type listInput struct {
	query.PageInput
	query.SortInput
	query.SearchInput
	query.DateRangeInput
	Name string `query:"name" maxLength:"100" doc:"Filter by name (case-insensitive contains)"`
}

type listOutput struct {
	Body struct {
		Items []OrgResponse `json:"items"`
		Total int64         `json:"total"`
	}
}

var sortableFields = []string{"name", "created_at", "updated_at"}

func (h *handler) list(_ context.Context, input *listInput) (*listOutput, error) {
	var orgs []Organization
	var total int64

	filters := query.NewFilterSet().
		Contains("name", input.Name)

	base := h.db.Model(&Organization{}).
		Scopes(
			filters.Apply(),
			query.WithSearch(input.SearchInput, "name", "description"),
			query.WithDateRange(input.DateRangeInput, "created_at"),
		)

	if err := base.Count(&total).Error; err != nil {
		return nil, huma.Error500InternalServerError("failed to count organizations")
	}

	err := base.
		Scopes(
			query.WithSort(input.SortInput, sortableFields...),
			query.WithPage(input.PageInput),
		).
		Find(&orgs).Error
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch organizations")
	}

	out := &listOutput{}
	out.Body.Total = total
	out.Body.Items = make([]OrgResponse, len(orgs))
	for i, o := range orgs {
		out.Body.Items[i] = toResponse(o)
	}
	return out, nil
}

type createInput struct {
	Body struct {
		Name        string `json:"name"        minLength:"1" maxLength:"100" doc:"Organization name"`
		Description string `json:"description" maxLength:"500"               doc:"Organization description"`
	}
}

type createOutput struct {
	Body OrgResponse
}

// create writes the organization row and a matching outbox event in a single DB
// transaction: core is the source of truth and is always consistent. The outbox
// worker provisions the organization in the identity provider afterwards, so the
// response carries synced=false until that delivery completes.
func (h *handler) create(ctx context.Context, input *createInput) (*createOutput, error) {
	org := Organization{
		ID:          uuid.NewString(),
		Name:        input.Body.Name,
		Description: input.Body.Description,
	}

	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		return outbox.Enqueue(tx, "organization", org.ID, EventOrgCreated, orgCreatedPayload{
			Name:        org.Name,
			Description: org.Description,
		})
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to create organization")
	}

	out := &createOutput{Body: toResponse(org)}
	return out, nil
}

// EventOrgCreated is the outbox event type emitted when an organization is created.
const EventOrgCreated = "organization.created"

type orgCreatedPayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// OrgProvisioner is the slice of the identity-provider client the outbox handler
// needs. Defined here so the domain package does not depend on a concrete client.
type OrgProvisioner interface {
	// FindByCoreID locates an already-provisioned organization by the core ID
	// stamped on its custom data, used to recover idempotently from a crash
	// between provisioning and the local commit. Returns found=false if none.
	FindByCoreID(ctx context.Context, coreID string) (externalID string, found bool, err error)
	// Create provisions the organization in the identity provider, stamping the
	// core ID on its custom data, and returns the external ID.
	Create(ctx context.Context, name, description, coreID string) (externalID string, err error)
}

// NewOutboxHandler returns an outbox.Handler that provisions organizations in the
// identity provider. It runs inside the worker's per-event transaction, so the
// external_id write and the event's completion commit atomically.
func NewOutboxHandler(p OrgProvisioner) outbox.Handler {
	return func(ctx context.Context, tx *gorm.DB, ev outbox.Event) error {
		switch ev.EventType {
		case EventOrgCreated:
			return provisionOrg(ctx, tx, p, ev)
		default:
			return huma.Error500InternalServerError("unknown outbox event type: " + ev.EventType)
		}
	}
}

func provisionOrg(ctx context.Context, tx *gorm.DB, p OrgProvisioner, ev outbox.Event) error {
	var org Organization
	if err := tx.First(&org, "id = ?", ev.AggregateID).Error; err != nil {
		return err
	}
	if org.ExternalID != nil && *org.ExternalID != "" {
		return nil // already provisioned — nothing to do
	}

	// Only the first attempt can safely skip the lookup: any retry may follow a
	// prior attempt that created the org in the provider but failed to commit
	// locally, so reconcile by core ID before creating a duplicate.
	if ev.Attempts > 0 {
		if extID, found, err := p.FindByCoreID(ctx, org.ID); err != nil {
			return err
		} else if found {
			return tx.Model(&org).Update("external_id", extID).Error
		}
	}

	extID, err := p.Create(ctx, org.Name, org.Description, org.ID)
	if err != nil {
		return err
	}
	return tx.Model(&org).Update("external_id", extID).Error
}
