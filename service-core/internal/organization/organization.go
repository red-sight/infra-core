package organization

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"gorm.io/gorm"

	"github.com/red-sight/infra-core/query"
)

type Organization struct {
	ID          string    `gorm:"primaryKey;type:uuid" json:"id"`
	LogtoOrgID  string    `gorm:"uniqueIndex;not null"  json:"logto_org_id"`
	Name        string    `gorm:"not null"              json:"name"`
	Description string    `gorm:"not null;default:''"   json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrgResponse struct {
	ID          string    `json:"id"           doc:"Internal organization ID"`
	LogtoOrgID  string    `json:"logto_org_id" doc:"Logto organization ID"`
	Name        string    `json:"name"         doc:"Organization name"`
	Description string    `json:"description"  doc:"Organization description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toResponse(o Organization) OrgResponse {
	return OrgResponse{
		ID:          o.ID,
		LogtoOrgID:  o.LogtoOrgID,
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
