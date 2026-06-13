package main

import (
	"encoding/json"
	"testing"

	"infra/registrator/internal/gateway"
	"infra/registrator/internal/registrar"
	"infra/registrator/internal/versiongate"
)

type fakeRenderer struct {
	data    []byte
	digests map[string]versiongate.Digest
}

func (f fakeRenderer) RenderWithDigests([]gateway.ServiceInfo, map[string][]string) ([]byte, map[string]versiongate.Digest, error) {
	return f.data, f.digests, nil
}

type fakeDeliverer struct {
	configs       []registrar.ConfigSummary
	serviceConfig []string
	created       []string
	updatedTo     string
}

func (f *fakeDeliverer) Configs(string) ([]registrar.ConfigSummary, error) { return f.configs, nil }

func (f *fakeDeliverer) CreateConfig(name string, _ []byte, labels map[string]string) (string, error) {
	f.created = append(f.created, name)
	cs := registrar.ConfigSummary{ID: "id-" + name}
	cs.Spec.Name = name
	cs.Spec.Labels = labels
	f.configs = append(f.configs, cs)
	return cs.ID, nil
}

func (f *fakeDeliverer) UpdateServiceConfig(_, _, _, configName string) error {
	f.updatedTo = configName
	return nil
}

func (f *fakeDeliverer) ServiceConfigNames(string) ([]string, error) { return f.serviceConfig, nil }

// seedCurrent builds a managed config object holding the given manifest and wires
// the fake service to mount it, simulating a prior delivery.
func seedCurrent(f *fakeDeliverer, name string, manifest map[string]versiongate.Digest) {
	b, _ := json.Marshal(manifest)
	cs := registrar.ConfigSummary{ID: "id-" + name}
	cs.Spec.Name = name
	cs.Spec.Labels = map[string]string{managedConfigLabel: "true", manifestLabel: string(b)}
	f.configs = append(f.configs, cs)
	f.serviceConfig = []string{name}
}

func TestDeliverArtifact(t *testing.T) {
	data := []byte(`{"version":3,"endpoints":[]}`)
	wantName := configNamePrefix + renderHash(data)

	t.Run("first delivery creates and rolls", func(t *testing.T) {
		r := fakeRenderer{data: data, digests: map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "h1"}}}
		d := &fakeDeliverer{}
		if err := deliverArtifact(r, d, nil, nil, "infra_krakend", "/etc/krakend/krakend.json"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.created) != 1 || d.created[0] != wantName {
			t.Errorf("created = %v, want [%s]", d.created, wantName)
		}
		if d.updatedTo != wantName {
			t.Errorf("service rolled to %q, want %q", d.updatedTo, wantName)
		}
	})

	t.Run("unchanged render is a no-op", func(t *testing.T) {
		digests := map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "h1"}}
		r := fakeRenderer{data: data, digests: digests}
		d := &fakeDeliverer{}
		seedCurrent(d, wantName, digests) // already delivered this exact render
		if err := deliverArtifact(r, d, nil, nil, "infra_krakend", "/x"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.created) != 0 || d.updatedTo != "" {
			t.Errorf("expected no-op, got created=%v updatedTo=%q", d.created, d.updatedTo)
		}
	})

	t.Run("contract change without version bump is rejected", func(t *testing.T) {
		r := fakeRenderer{data: data, digests: map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "NEW"}}}
		d := &fakeDeliverer{}
		seedCurrent(d, configNamePrefix+"old", map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "OLD"}})
		err := deliverArtifact(r, d, nil, nil, "infra_krakend", "/x")
		if err == nil {
			t.Fatal("expected version-gate rejection, got nil")
		}
		if len(d.created) != 0 || d.updatedTo != "" {
			t.Errorf("rejected delivery must not create/roll, got created=%v updatedTo=%q", d.created, d.updatedTo)
		}
	})

	t.Run("contract change with version bump is delivered", func(t *testing.T) {
		r := fakeRenderer{data: data, digests: map[string]versiongate.Digest{"svc-a": {Version: "1.1.0", ContractHash: "NEW"}}}
		d := &fakeDeliverer{}
		seedCurrent(d, configNamePrefix+"old", map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "OLD"}})
		if err := deliverArtifact(r, d, nil, nil, "infra_krakend", "/x"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(d.created) != 1 || d.created[0] != wantName || d.updatedTo != wantName {
			t.Errorf("expected delivery of %s, got created=%v updatedTo=%q", wantName, d.created, d.updatedTo)
		}
	})
}
