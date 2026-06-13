package main

import "testing"

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
