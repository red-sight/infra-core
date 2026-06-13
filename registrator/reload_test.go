package main

import (
	"testing"

	"infra/registrator/internal/registrar"
)

type fakeAdapter struct{ events []registrar.ServiceEvent }

func (fakeAdapter) WatchServices() <-chan registrar.ServiceEvent { return nil }
func (f fakeAdapter) Scan() []registrar.ServiceEvent             { return f.events }
func (fakeAdapter) Mode() string                                 { return "test" }

func TestScanOnce(t *testing.T) {
	adapter := fakeAdapter{events: []registrar.ServiceEvent{
		{Name: "a", Port: 3000, Healthy: true},
		{Name: "b", Port: 3001, Healthy: true},
		{Name: "gone", Removed: true},  // removed → skip
		{Name: "sick", Healthy: false}, // unhealthy → skip
	}}
	reg := registrar.NewRegistry()

	if n := scanOnce(adapter, reg); n != 2 {
		t.Fatalf("scanOnce count = %d, want 2", n)
	}
	names := map[string]bool{}
	for _, s := range reg.Services() {
		names[s.Name] = true
	}
	if !names["a"] || !names["b"] {
		t.Errorf("expected healthy services a and b, got %v", names)
	}
	if names["gone"] || names["sick"] {
		t.Errorf("removed/unhealthy services must be excluded, got %v", names)
	}
}

func TestResolveReloadMode(t *testing.T) {
	tests := []struct {
		name      string
		envVal    string // INFRA_REGISTRATOR_RELOAD_MODE; "" means unset
		dockerMod string
		want      reloadMode
	}{
		{"explicit auto wins over docker mode", "auto", "swarm", modeAuto},
		{"explicit artifact wins over docker mode", "artifact", "compose", modeArtifact},
		{"unset infers artifact from swarm", "", "swarm", modeArtifact},
		{"unset infers auto from compose", "", "compose", modeAuto},
		{"unset with unknown docker mode defaults to auto", "", "", modeAuto},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("INFRA_REGISTRATOR_RELOAD_MODE", tt.envVal)
			if got := resolveReloadMode(tt.dockerMod); got != tt.want {
				t.Errorf("resolveReloadMode(%q) with env=%q = %q, want %q", tt.dockerMod, tt.envVal, got, tt.want)
			}
		})
	}
}
