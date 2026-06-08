package query

import (
	"strings"

	"gorm.io/gorm"
)

type filterEntry struct {
	apply func(*gorm.DB) *gorm.DB
}

// FilterSet accumulates field-level filter conditions.
// Each method is a noop when the provided value is the zero value for its type.
// Build the GORM scope with Apply().
type FilterSet struct {
	entries []filterEntry
}

func NewFilterSet() *FilterSet { return &FilterSet{} }

// Exact adds WHERE column = value (case-sensitive).
// Noop when value is empty.
func (f *FilterSet) Exact(column, value string) *FilterSet {
	if value != "" {
		f.entries = append(f.entries, filterEntry{func(db *gorm.DB) *gorm.DB {
			return db.Where(column+" = ?", value)
		}})
	}
	return f
}

// IExact adds WHERE LOWER(column) = LOWER(value).
// Noop when value is empty.
func (f *FilterSet) IExact(column, value string) *FilterSet {
	if value != "" {
		f.entries = append(f.entries, filterEntry{func(db *gorm.DB) *gorm.DB {
			return db.Where("LOWER("+column+") = LOWER(?)", value)
		}})
	}
	return f
}

// Contains adds WHERE column ILIKE %value% (case-insensitive).
// Noop when value is empty.
func (f *FilterSet) Contains(column, value string) *FilterSet {
	if value != "" {
		pattern := "%" + value + "%"
		f.entries = append(f.entries, filterEntry{func(db *gorm.DB) *gorm.DB {
			return db.Where(column+" ILIKE ?", pattern)
		}})
	}
	return f
}

// In adds WHERE column IN (values).
// Noop when values is empty or all elements are empty strings.
func (f *FilterSet) In(column string, values []string) *FilterSet {
	filtered := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) > 0 {
		f.entries = append(f.entries, filterEntry{func(db *gorm.DB) *gorm.DB {
			return db.Where(column+" IN ?", filtered)
		}})
	}
	return f
}

// InCSV parses a comma-separated string and applies an IN filter.
// Noop when value is empty.
func (f *FilterSet) InCSV(column, value string) *FilterSet {
	if value == "" {
		return f
	}
	parts := strings.Split(value, ",")
	return f.In(column, parts)
}

// Apply returns a GORM scope that applies all accumulated filters.
func (f *FilterSet) Apply() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		for _, e := range f.entries {
			db = e.apply(db)
		}
		return db
	}
}
