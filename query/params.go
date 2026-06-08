package query

import "time"

// PageInput provides standard pagination params. Embed in Huma input structs.
type PageInput struct {
	Page     int `query:"page"      default:"1"  minimum:"1"             doc:"Page number (1-based)"`
	PageSize int `query:"page_size" default:"20" minimum:"1" maximum:"100" doc:"Items per page"`
}

// SortInput provides sort params. Use with WithSort(db, input.SortInput, allowedFields...).
// Empty SortBy means no explicit ordering (caller's default applies).
type SortInput struct {
	SortBy  string `query:"sort_by"                                doc:"Column to sort by (allowed: see endpoint description)"`
	SortDir string `query:"sort_dir" default:"desc" enum:"asc,desc" doc:"Sort direction"`
}

// SearchInput provides a case-insensitive substring search param.
// Empty Q means no search filter applied.
type SearchInput struct {
	Q string `query:"q" maxLength:"200" doc:"Substring search (case-insensitive, applied across searchable fields)"`
}

// DateRangeInput provides an inclusive date-time range filter.
// Zero value means the bound is open (not applied).
// Use with WithDateRange(db, input.DateRangeInput, "column_name").
type DateRangeInput struct {
	DateFrom time.Time `query:"date_from" doc:"Range start — inclusive, RFC 3339 (e.g. 2024-01-01T00:00:00Z). Omit for open start."`
	DateTo   time.Time `query:"date_to"   doc:"Range end — inclusive, RFC 3339 (e.g. 2024-12-31T23:59:59Z). Omit for open end."`
}

// ErrRequired is returned by Require when a param was not provided.
type ErrRequired struct {
	Field string
}

func (e ErrRequired) Error() string { return "required parameter missing: " + e.Field }

// Require asserts that a string param was provided (non-empty).
func RequireString(field, v string) error {
	if v == "" {
		return ErrRequired{Field: field}
	}
	return nil
}

// RequireTime asserts that a time param was provided (non-zero).
func RequireTime(field string, v time.Time) error {
	if v.IsZero() {
		return ErrRequired{Field: field}
	}
	return nil
}
