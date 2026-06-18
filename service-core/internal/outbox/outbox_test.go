package outbox

import (
	"testing"
	"time"
)

func TestBackoffExponentialAndCap(t *testing.T) {
	w := NewWorker(nil, nil, Config{BaseBackoff: time.Second, MaxBackoff: 8 * time.Second, MaxAttempts: 100})

	cases := map[int]time.Duration{
		1: 1 * time.Second,
		2: 2 * time.Second,
		3: 4 * time.Second,
		4: 8 * time.Second, // hits cap
		5: 8 * time.Second, // stays capped
		9: 8 * time.Second,
	}
	for attempts, want := range cases {
		if got := w.backoff(attempts); got != want {
			t.Errorf("backoff(%d) = %s, want %s", attempts, got, want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	w := NewWorker(nil, nil, Config{})
	c := w.cfg
	if c.PollInterval <= 0 || c.BatchSize <= 0 || c.MaxAttempts <= 0 || c.BaseBackoff <= 0 || c.MaxBackoff <= 0 {
		t.Fatalf("zero config did not get defaults: %+v", c)
	}
}

func TestEnqueueMarshalsPayload(t *testing.T) {
	// Enqueue marshals the payload before touching the DB; an unmarshalable value
	// must fail fast rather than persist a broken row. nil tx is never reached.
	err := Enqueue(nil, "organization", "id", "organization.created", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error for non-serializable payload")
	}
}
