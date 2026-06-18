// Package outbox implements the transactional outbox pattern: a write to a local
// aggregate and the intent to propagate it to an external system are persisted in
// the same DB transaction, and a background worker delivers that intent with
// retries until it succeeds. This keeps the local store authoritative and
// consistent while the external system converges — no periodic reconciliation.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Event statuses.
const (
	StatusPending = "pending"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// Event is a row in outbox_events.
type Event struct {
	ID            string          `gorm:"primaryKey;type:uuid"`
	Aggregate     string          `gorm:"not null"`
	AggregateID   string          `gorm:"type:uuid;not null"`
	EventType     string          `gorm:"not null"`
	Payload       json.RawMessage `gorm:"type:jsonb;not null"`
	Status        string          `gorm:"not null;default:pending"`
	Attempts      int             `gorm:"not null;default:0"`
	LastError     string          `gorm:"not null;default:''"`
	NextAttemptAt time.Time       `gorm:"not null"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (Event) TableName() string { return "outbox_events" }

// Enqueue persists an event inside the caller's transaction. Call it in the same
// tx that writes the aggregate so the two commit atomically.
func Enqueue(tx *gorm.DB, aggregate, aggregateID, eventType string, payload any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return tx.Create(&Event{
		ID:            uuid.NewString(),
		Aggregate:     aggregate,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       raw,
		Status:        StatusPending,
		NextAttemptAt: time.Now(),
	}).Error
}

// Handler delivers a single event. It runs inside the worker's per-event
// transaction (tx), so any aggregate writes it makes commit atomically with the
// event being marked done. Returning an error schedules a retry; the handler must
// not leave partial aggregate writes when it returns an error.
type Handler func(ctx context.Context, tx *gorm.DB, ev Event) error

// Config tunes the worker. Zero values fall back to sensible defaults.
type Config struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	BaseBackoff  time.Duration
	MaxBackoff   time.Duration
}

func (c *Config) withDefaults() {
	if c.PollInterval <= 0 {
		c.PollInterval = 2 * time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 20
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 10
	}
	if c.BaseBackoff <= 0 {
		c.BaseBackoff = 5 * time.Second
	}
	if c.MaxBackoff <= 0 {
		c.MaxBackoff = 5 * time.Minute
	}
}

// Worker polls for due events and delivers them via the handler.
type Worker struct {
	db      *gorm.DB
	handler Handler
	cfg     Config
}

func NewWorker(db *gorm.DB, handler Handler, cfg Config) *Worker {
	cfg.withDefaults()
	return &Worker{db: db, handler: handler, cfg: cfg}
}

// Run blocks until ctx is cancelled, draining due events each tick.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("outbox worker started (poll=%s, max_attempts=%d)", w.cfg.PollInterval, w.cfg.MaxAttempts)
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("outbox worker stopped")
			return
		case <-ticker.C:
			// Drain up to BatchSize due events per tick; stop early when none remain.
			for i := 0; i < w.cfg.BatchSize; i++ {
				done, err := w.processOne(ctx)
				if err != nil {
					log.Printf("outbox: process error: %v", err)
					break
				}
				if done {
					break // no due events
				}
			}
		}
	}
}

var errNoWork = errors.New("no due events")

// processOne locks and delivers the next due event in its own transaction.
// Returns done=true when there is nothing to do.
func (w *Worker) processOne(ctx context.Context) (done bool, err error) {
	txErr := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var ev Event
		// SKIP LOCKED lets multiple workers/replicas process disjoint events safely.
		lookupErr := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND next_attempt_at <= ?", StatusPending, time.Now()).
			Order("next_attempt_at").
			First(&ev).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return errNoWork
		}
		if lookupErr != nil {
			return lookupErr
		}

		// The handler runs in this tx: on success its aggregate writes and the
		// event's completion commit together.
		if hErr := w.handler(ctx, tx, ev); hErr != nil {
			ev.Attempts++
			ev.LastError = hErr.Error()
			if ev.Attempts >= w.cfg.MaxAttempts {
				ev.Status = StatusFailed
				log.Printf("outbox: event %s (%s) failed permanently after %d attempts: %v", ev.ID, ev.EventType, ev.Attempts, hErr)
			} else {
				ev.NextAttemptAt = time.Now().Add(w.backoff(ev.Attempts))
				log.Printf("outbox: event %s (%s) attempt %d failed, retrying: %v", ev.ID, ev.EventType, ev.Attempts, hErr)
			}
			// Commit the bookkeeping (not a rollback) so attempts/backoff persist.
			return tx.Save(&ev).Error
		}

		ev.Status = StatusDone
		ev.LastError = ""
		return tx.Save(&ev).Error
	})

	if errors.Is(txErr, errNoWork) {
		return true, nil
	}
	return false, txErr
}

// backoff returns an exponential delay capped at MaxBackoff.
func (w *Worker) backoff(attempts int) time.Duration {
	d := w.cfg.BaseBackoff
	for i := 1; i < attempts; i++ {
		d *= 2
		if d >= w.cfg.MaxBackoff {
			return w.cfg.MaxBackoff
		}
	}
	return d
}
