package registrar

import (
	"os"
	"os/exec"
	"slices"
	"testing"
)

// TestConfigDeliverySwarm exercises the Swarm config-object delivery primitives
// against a real active Swarm: create a config, point a service at it, then swap
// the service onto a second config via UpdateServiceConfig. Skipped unless
// INFRA_TEST_SWARM is set (needs `docker swarm init`). The test service is
// created/removed with the docker CLI — creating services is not a registrator
// responsibility, so it stays out of DockerClient.
func TestConfigDeliverySwarm(t *testing.T) {
	if os.Getenv("INFRA_TEST_SWARM") == "" {
		t.Skip("set INFRA_TEST_SWARM=1 to run (needs an active single-node swarm)")
	}

	const (
		svc    = "itest-krakend-svc"
		cfgA   = "itest-cfg-a"
		cfgB   = "itest-cfg-b"
		target = "/etc/krakend/krakend.json"
		label  = "infra.itest=1"
	)
	d := NewDockerClient()

	// Clean slate, and always clean up afterwards.
	cleanup := func() {
		_ = exec.Command("docker", "service", "rm", svc).Run()
		for _, c := range []string{cfgA, cfgB} {
			_ = exec.Command("docker", "config", "rm", c).Run()
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	// 1. Create the first config object (method under test).
	idA, err := d.CreateConfig(cfgA, []byte(`{"version":3}`), map[string]string{"infra.itest": "1"})
	if err != nil {
		t.Fatalf("CreateConfig A: %v", err)
	}

	// 2. Create a service mounting config A (setup via CLI).
	out, err := exec.Command("docker", "service", "create",
		"--name", svc,
		"--config", "source="+cfgA+",target="+target,
		"alpine", "sleep", "600",
	).CombinedOutput()
	if err != nil {
		t.Fatalf("docker service create: %v\n%s", err, out)
	}

	if names, _ := d.serviceConfigNames(svc); !slices.Contains(names, cfgA) {
		t.Fatalf("service should mount %s, got %v", cfgA, names)
	}

	// 3. Create the second config and swap the service onto it (method under test).
	idB, err := d.CreateConfig(cfgB, []byte(`{"version":3,"name":"v2"}`), map[string]string{"infra.itest": "1"})
	if err != nil {
		t.Fatalf("CreateConfig B: %v", err)
	}
	if idB == idA {
		t.Fatal("distinct content must yield distinct config IDs")
	}

	if err := d.UpdateServiceConfig(svc, target, idB, cfgB); err != nil {
		t.Fatalf("UpdateServiceConfig: %v", err)
	}

	// 4. The service now references config B, not A.
	names, err := d.serviceConfigNames(svc)
	if err != nil {
		t.Fatalf("serviceConfigNames: %v", err)
	}
	if !slices.Contains(names, cfgB) || slices.Contains(names, cfgA) {
		t.Fatalf("after swap expected only %s, got %v", cfgB, names)
	}

	// 5. Both configs are discoverable by label.
	configs, err := d.Configs(label)
	if err != nil {
		t.Fatalf("Configs: %v", err)
	}
	found := map[string]bool{}
	for _, c := range configs {
		found[c.Spec.Name] = true
	}
	if !found[cfgA] || !found[cfgB] {
		t.Errorf("Configs(%q) should list both configs, got %v", label, found)
	}
}
