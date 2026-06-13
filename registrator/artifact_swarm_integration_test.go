package main

import (
	"os"
	"os/exec"
	"slices"
	"testing"

	"infra/registrator/internal/registrar"
	"infra/registrator/internal/versiongate"
)

// TestDeliverArtifactSwarm exercises the full artifact delivery path against a
// real swarm: a real DockerClient as the deliverer and a fake renderer for the
// config bytes/digests. It verifies first delivery, idempotency, the version
// gate, and re-delivery after a version bump — all through real Swarm config
// objects and a real service update. Skipped unless INFRA_TEST_SWARM is set.
func TestDeliverArtifactSwarm(t *testing.T) {
	if os.Getenv("INFRA_TEST_SWARM") == "" {
		t.Skip("set INFRA_TEST_SWARM=1 to run (needs an active single-node swarm)")
	}

	const (
		svc    = "itest-krakend"
		target = "/etc/krakend/krakend.json"
	)
	d := registrar.NewDockerClient()

	cleanup := func() {
		_ = exec.Command("docker", "service", "rm", svc).Run()
		if configs, err := d.Configs(managedConfigLabel + "=true"); err == nil {
			for _, c := range configs {
				_ = d.RemoveConfig(c.ID)
			}
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	if out, err := exec.Command("docker", "service", "create", "--name", svc, "alpine", "sleep", "600").CombinedOutput(); err != nil {
		t.Fatalf("create service: %v\n%s", err, out)
	}

	dataV1 := []byte(`{"version":3,"endpoints":[],"v":1}`)
	nameV1 := configNamePrefix + renderHash(dataV1)
	digestsV1 := map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "h1"}}

	// 1. First delivery: creates the config object and rolls the service onto it.
	if err := deliverArtifact(fakeRenderer{data: dataV1, digests: digestsV1}, d, nil, nil, svc, target); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if names, _ := d.ServiceConfigNames(svc); !slices.Contains(names, nameV1) {
		t.Fatalf("service should mount %s, got %v", nameV1, names)
	}
	if n := countManaged(t, d); n != 1 {
		t.Fatalf("expected 1 managed config, got %d", n)
	}

	// 2. Same render again → idempotent no-op (no new config object).
	if err := deliverArtifact(fakeRenderer{data: dataV1, digests: digestsV1}, d, nil, nil, svc, target); err != nil {
		t.Fatalf("idempotent delivery: %v", err)
	}
	if n := countManaged(t, d); n != 1 {
		t.Errorf("idempotent delivery must not create a config; managed count = %d", n)
	}

	// 3. Contract changed but version not bumped → rejected, service unchanged.
	badData := []byte(`{"version":3,"endpoints":[],"v":2}`)
	badDigests := map[string]versiongate.Digest{"svc-a": {Version: "1.0.0", ContractHash: "h2"}}
	if err := deliverArtifact(fakeRenderer{data: badData, digests: badDigests}, d, nil, nil, svc, target); err == nil {
		t.Error("expected version-gate rejection, got nil")
	}
	if names, _ := d.ServiceConfigNames(svc); !slices.Contains(names, nameV1) {
		t.Errorf("rejected delivery must leave service on %s, got %v", nameV1, names)
	}

	// 4. Contract changed with a version bump → delivered, service rolled over.
	dataV2 := []byte(`{"version":3,"endpoints":[],"v":3}`)
	nameV2 := configNamePrefix + renderHash(dataV2)
	goodDigests := map[string]versiongate.Digest{"svc-a": {Version: "1.1.0", ContractHash: "h2"}}
	if err := deliverArtifact(fakeRenderer{data: dataV2, digests: goodDigests}, d, nil, nil, svc, target); err != nil {
		t.Fatalf("bumped delivery: %v", err)
	}
	names, _ := d.ServiceConfigNames(svc)
	if !slices.Contains(names, nameV2) || slices.Contains(names, nameV1) {
		t.Errorf("service should now mount only %s, got %v", nameV2, names)
	}
}

func countManaged(t *testing.T, d *registrar.DockerClient) int {
	t.Helper()
	configs, err := d.Configs(managedConfigLabel + "=true")
	if err != nil {
		t.Fatalf("list managed configs: %v", err)
	}
	return len(configs)
}
