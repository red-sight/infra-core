package organization

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
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
	Slug        string    `gorm:"uniqueIndex;not null"  json:"slug"`
	Name        string    `gorm:"not null"              json:"name"`
	Description string    `gorm:"not null;default:''"   json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrgResponse struct {
	ID          string    `json:"id"          doc:"Internal organization ID"`
	ExternalID  *string   `json:"external_id" doc:"Identity-provider organization ID (Logto). Null until synced."`
	Synced      bool      `json:"synced"      doc:"Whether the organization has been provisioned in the identity provider."`
	Slug        string    `json:"slug"        doc:"DNS-safe label addressing the tenant frontend subdomain"`
	Name        string    `json:"name"        doc:"Organization name"`
	Description string    `json:"description" doc:"Organization description"`
	Domain      string    `json:"domain"      doc:"Frontend hostname the organization is served on (slug-derived; apex for the master org)"`
	IsMaster    bool      `json:"is_master"   doc:"True if this is the master organization (apex domain, empty slug)"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// domainFor derives the organization's frontend hostname. The master org (empty
// slug) lives on the apex domain; everyone else on a "<slug>.<base>" subdomain.
// Mirrors the origin derivation in provisionOrg, without the protocol.
func (h *handler) domainFor(slug string) string {
	if slug == "" {
		return h.baseDomain
	}
	return slug + "." + h.baseDomain
}

func (h *handler) toResponse(o Organization) OrgResponse {
	return OrgResponse{
		ID:          o.ID,
		ExternalID:  o.ExternalID,
		Synced:      o.ExternalID != nil && *o.ExternalID != "",
		Slug:        o.Slug,
		Name:        o.Name,
		Description: o.Description,
		Domain:      h.domainFor(o.Slug),
		IsMaster:    o.Slug == "",
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
	}
}

// slugPattern mirrors the create-input validation; DNS-label-safe, 2–40 chars.
var slugPattern = regexp.MustCompile(`^[a-z0-9-]{2,40}$`)

// reservedSlugs are subdomains used by platform services — they must never be
// claimable as a tenant slug or routing would collide.
var reservedSlugs = map[string]bool{
	"admin": true, "auth": true, "auth-admin": true, "traefik": true,
	"api": true, "app": true, "www": true,
}

// validateSlug enforces format, no leading/trailing hyphen, and the reserved list.
func validateSlug(slug string) error {
	if !slugPattern.MatchString(slug) {
		return errors.New("slug must be 2–40 chars of lowercase letters, digits or hyphens")
	}
	if slug[0] == '-' || slug[len(slug)-1] == '-' {
		return errors.New("slug must not start or end with a hyphen")
	}
	if reservedSlugs[slug] {
		return errors.New("slug is reserved")
	}
	return nil
}

func RegisterRoutes(api huma.API, db *gorm.DB, baseDomain string, dir Directory) {
	h := &handler{db: db, baseDomain: baseDomain, dir: dir}

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
		OperationID: "get-organization",
		Method:      http.MethodGet,
		Path:        "/admin/organizations/{id}",
		Summary:     "Get organization",
		Description: "Returns a single organization by its internal ID.",
		Tags:        []string{"Admin"},
		Extensions: map[string]any{
			"x-infra-protected": true,
			"x-infra-scopes":    []string{"read:organizations"},
		},
	}, h.get)

	huma.Register(api, huma.Operation{
		OperationID: "list-organization-members",
		Method:      http.MethodGet,
		Path:        "/admin/organizations/{id}/members",
		Summary:     "List organization members",
		Description: "Returns the organization's members and their org-scoped roles, sourced from the identity provider. Paginated; q is a case-insensitive match on name/email. An unprovisioned organization has no members yet and returns an empty list.",
		Tags:        []string{"Admin"},
		Extensions: map[string]any{
			"x-infra-protected": true,
			"x-infra-scopes":    []string{"read:organizations"},
		},
	}, h.listMembers)

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

	huma.Register(api, huma.Operation{
		OperationID: "get-tenant-by-host",
		Method:      http.MethodGet,
		Path:        "/tenant/by-host",
		Summary:     "Resolve tenant by host",
		Description: "Public endpoint the tenant frontend calls (before login) to resolve its organization from its hostname. The apex domain maps to the master organization; a subdomain maps to the org with that slug.",
		Tags:        []string{"Tenant"},
		Extensions: map[string]any{
			"x-infra-protected": false,
		},
	}, h.getByHost)
}

type handler struct {
	db         *gorm.DB
	baseDomain string
	dir        Directory
}

// MemberRole is an org-scoped role assigned to a member.
type MemberRole struct {
	ID   string `json:"id"   doc:"Role ID"`
	Name string `json:"name" doc:"Role name"`
}

// Member is a user belonging to an organization, with their org-scoped roles.
type Member struct {
	ID     string       `json:"id"     doc:"User ID"`
	Name   string       `json:"name"   doc:"Display name (falls back to username)"`
	Email  string       `json:"email"  doc:"Primary email"`
	Avatar string       `json:"avatar" doc:"Avatar URL (may be empty)"`
	Roles  []MemberRole `json:"roles"  doc:"Org-scoped roles held in this organization"`
}

// Directory reads organization membership from the identity provider. Defined here
// (returning provider-neutral types) so the domain package does not depend on a
// concrete client; the Logto client implements it.
type Directory interface {
	ListMembers(ctx context.Context, orgExternalID string, page, pageSize int, q string) (members []Member, total int, err error)
}

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
		out.Body.Items[i] = h.toResponse(o)
	}
	return out, nil
}

type createInput struct {
	Body struct {
		Name        string `json:"name"                  minLength:"1" maxLength:"100"   doc:"Organization name"`
		Slug        string `json:"slug"                  pattern:"^[a-z0-9-]{2,40}$"     doc:"DNS-safe subdomain label (e.g. acme)"`
		Description string `json:"description,omitempty" maxLength:"500"                 doc:"Organization description (optional)"`
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
	if err := validateSlug(input.Body.Slug); err != nil {
		return nil, huma.Error422UnprocessableEntity(err.Error())
	}

	org := Organization{
		ID:          uuid.NewString(),
		Slug:        input.Body.Slug,
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
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, huma.Error409Conflict("slug already taken")
		}
		return nil, huma.Error500InternalServerError("failed to create organization")
	}

	out := &createOutput{Body: h.toResponse(org)}
	return out, nil
}

type getInput struct {
	ID string `path:"id" doc:"Internal organization ID"`
}

type getOutput struct {
	Body OrgResponse
}

// get returns a single organization by its internal ID.
func (h *handler) get(ctx context.Context, input *getInput) (*getOutput, error) {
	// Guard the UUID format before hitting the DB so a malformed ID is a clean
	// 404 rather than a Postgres "invalid input syntax for type uuid" 500.
	if _, err := uuid.Parse(input.ID); err != nil {
		return nil, huma.Error404NotFound("organization not found")
	}

	var org Organization
	err := h.db.WithContext(ctx).First(&org, "id = ?", input.ID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("organization not found")
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch organization")
	}

	return &getOutput{Body: h.toResponse(org)}, nil
}

type listMembersInput struct {
	ID string `path:"id" doc:"Internal organization ID"`
	query.PageInput
	query.SearchInput
}

type listMembersOutput struct {
	Body struct {
		Items []Member `json:"items"`
		Total int      `json:"total"`
	}
}

// listMembers proxies the organization's membership from the identity provider.
// Membership lives in Logto, not core, so this reads through the Directory rather
// than the DB. An organization not yet provisioned (no external_id) has no members
// there and returns an empty page rather than an error.
func (h *handler) listMembers(ctx context.Context, input *listMembersInput) (*listMembersOutput, error) {
	if _, err := uuid.Parse(input.ID); err != nil {
		return nil, huma.Error404NotFound("organization not found")
	}

	var org Organization
	err := h.db.WithContext(ctx).First(&org, "id = ?", input.ID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("organization not found")
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to fetch organization")
	}

	out := &listMembersOutput{}
	out.Body.Items = []Member{}
	if org.ExternalID == nil || *org.ExternalID == "" {
		return out, nil
	}

	members, total, err := h.dir.ListMembers(ctx, *org.ExternalID, input.Page, input.PageSize, input.Q)
	if err != nil {
		return nil, huma.Error502BadGateway("failed to list organization members")
	}
	if members != nil {
		out.Body.Items = members
	}
	out.Body.Total = total
	return out, nil
}

type getByHostInput struct {
	Host string `query:"host" required:"true" doc:"Frontend hostname, e.g. acme.app.localhost or the apex app.localhost"`
}

type tenantOutput struct {
	Body struct {
		ID       string `json:"id"        doc:"Internal organization ID"`
		Slug     string `json:"slug"      doc:"Tenant slug (empty for the master organization)"`
		Name     string `json:"name"      doc:"Organization name"`
		IsMaster bool   `json:"is_master" doc:"True if this is the master organization (apex domain)"`
	}
}

// getByHost resolves an organization from its frontend hostname. Public (no auth):
// the tenant frontend calls it before login. The apex domain (== base domain) maps
// to the master organization (empty slug); "<slug>.<base>" maps to that slug.
func (h *handler) getByHost(ctx context.Context, input *getByHostInput) (*tenantOutput, error) {
	host := strings.ToLower(input.Host)
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i] // strip any port
	}

	var slug string
	switch {
	case host == h.baseDomain:
		slug = "" // apex → master organization
	case strings.HasSuffix(host, "."+h.baseDomain):
		label := strings.TrimSuffix(host, "."+h.baseDomain)
		if label == "" || strings.Contains(label, ".") {
			return nil, huma.Error404NotFound("unknown tenant host")
		}
		slug = label
	default:
		return nil, huma.Error404NotFound("unknown tenant host")
	}

	var org Organization
	err := h.db.WithContext(ctx).First(&org, "slug = ?", slug).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, huma.Error404NotFound("organization not found")
	}
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to resolve organization")
	}

	out := &tenantOutput{}
	out.Body.ID = org.ID
	out.Body.Slug = org.Slug
	out.Body.Name = org.Name
	out.Body.IsMaster = org.Slug == ""
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
	// EnsureTenantRedirectURI registers the tenant frontend's OIDC redirect URIs
	// for the given origin on the shared Tenant application. Idempotent.
	EnsureTenantRedirectURI(ctx context.Context, origin string) error
}

// HandlerConfig carries the deployment URL shape needed to derive a tenant's
// frontend origin (<protocol>://<slug>.<baseDomain>).
type HandlerConfig struct {
	HTTPProtocol string
	BaseDomain   string
}

// NewOutboxHandler returns an outbox.Handler that provisions organizations in the
// identity provider. It runs inside the worker's per-event transaction, so the
// external_id write and the event's completion commit atomically.
func NewOutboxHandler(p OrgProvisioner, cfg HandlerConfig) outbox.Handler {
	return func(ctx context.Context, tx *gorm.DB, ev outbox.Event) error {
		switch ev.EventType {
		case EventOrgCreated:
			return provisionOrg(ctx, tx, p, cfg, ev)
		default:
			return huma.Error500InternalServerError("unknown outbox event type: " + ev.EventType)
		}
	}
}

func provisionOrg(ctx context.Context, tx *gorm.DB, p OrgProvisioner, cfg HandlerConfig, ev outbox.Event) error {
	var org Organization
	if err := tx.First(&org, "id = ?", ev.AggregateID).Error; err != nil {
		return err
	}

	// Step 1: ensure the organization exists in the identity provider (external_id).
	if org.ExternalID == nil || *org.ExternalID == "" {
		var extID string
		// Only retries can safely look up: a prior attempt may have created the org
		// but failed to commit locally — reconcile by core ID before duplicating.
		if ev.Attempts > 0 {
			id, found, err := p.FindByCoreID(ctx, org.ID)
			if err != nil {
				return err
			}
			if found {
				extID = id
			}
		}
		if extID == "" {
			id, err := p.Create(ctx, org.Name, org.Description, org.ID)
			if err != nil {
				return err
			}
			extID = id
		}
		if err := tx.Model(&org).Update("external_id", extID).Error; err != nil {
			return err
		}
	}

	// Step 2: ensure the tenant frontend's redirect URI is registered. Idempotent,
	// re-run on every retry until it succeeds. The master organization (empty slug)
	// lives on the apex domain; everyone else on a subdomain.
	var origin string
	if org.Slug == "" {
		origin = fmt.Sprintf("%s://%s", cfg.HTTPProtocol, cfg.BaseDomain)
	} else {
		origin = fmt.Sprintf("%s://%s.%s", cfg.HTTPProtocol, org.Slug, cfg.BaseDomain)
	}
	return p.EnsureTenantRedirectURI(ctx, origin)
}

// EnsureMaster seeds the master organization (the product owner's org, hosted on
// the apex domain) if it does not yet exist. The master is the row with an empty
// slug; UNIQUE(slug) guarantees there is at most one. Idempotent and safe under
// concurrent startups (a lost race surfaces as a duplicate-key, treated as done).
// Like a normal create, it enqueues an outbox event so the org is provisioned in
// the identity provider with the apex redirect URI.
func EnsureMaster(db *gorm.DB, name string) error {
	var count int64
	if err := db.Model(&Organization{}).Where("slug = ?", "").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	org := Organization{ID: uuid.NewString(), Slug: "", Name: name}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&org).Error; err != nil {
			return err
		}
		return outbox.Enqueue(tx, "organization", org.ID, EventOrgCreated, orgCreatedPayload{Name: org.Name})
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil // another replica seeded it first
	}
	return err
}
