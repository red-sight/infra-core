package query

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// WithPage applies offset/limit derived from PageInput.
func WithPage(p PageInput) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset((p.Page - 1) * p.PageSize).Limit(p.PageSize)
	}
}

// WithSort applies ORDER BY from SortInput.
// allowed is the whitelist of column names; an unrecognised SortBy is silently ignored.
func WithSort(s SortInput, allowed ...string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if s.SortBy == "" {
			return db
		}
		for _, a := range allowed {
			if a == s.SortBy {
				dir := "DESC"
				if strings.EqualFold(s.SortDir, "asc") {
					dir = "ASC"
				}
				return db.Order(fmt.Sprintf("%s %s", s.SortBy, dir))
			}
		}
		return db
	}
}

// WithSearch applies an ILIKE substring filter across the given columns joined by OR.
// Noop when Q is empty or no columns are provided.
func WithSearch(s SearchInput, columns ...string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if s.Q == "" || len(columns) == 0 {
			return db
		}
		pattern := "%" + s.Q + "%"
		clauses := make([]string, len(columns))
		args := make([]any, len(columns))
		for i, col := range columns {
			clauses[i] = col + " ILIKE ?"
			args[i] = pattern
		}
		return db.Where("("+strings.Join(clauses, " OR ")+")", args...)
	}
}

// WithDateRange applies an inclusive date-time range filter on the given column.
// Zero-value bounds are treated as open (not applied).
func WithDateRange(d DateRangeInput, column string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !d.DateFrom.IsZero() {
			db = db.Where(column+" >= ?", d.DateFrom)
		}
		if !d.DateTo.IsZero() {
			db = db.Where(column+" <= ?", d.DateTo)
		}
		return db
	}
}
