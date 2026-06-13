package registrar

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunOneShotKrakendCheck exercises the one-shot container plumbing against a
// real Docker daemon, running `krakend check` on a good and a bad config. It is
// skipped unless INFRA_TEST_DOCKER is set, since it needs Docker and the krakend
// image. SelfBindSource is not exercised here (it only resolves inside a container);
// this test binds an explicit host dir instead.
func TestRunOneShotKrakendCheck(t *testing.T) {
	if os.Getenv("INFRA_TEST_DOCKER") == "" {
		t.Skip("set INFRA_TEST_DOCKER=1 to run (needs Docker + krakend image)")
	}

	image := os.Getenv("INFRA_KRAKEND_IMAGE")
	if image == "" {
		image = "krakend:latest"
	}

	d := NewDockerClient()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "krakend.json")

	run := func() OneShotResult {
		t.Helper()
		res, err := d.RunOneShot(OneShotSpec{
			Name:        "krakend-check-test",
			Image:       image,
			Cmd:         []string{"check", "-c", "/cfg/krakend.json"},
			Binds:       []string{dir + ":/cfg:ro"},
			NetworkNone: true,
		})
		if err != nil {
			t.Fatalf("RunOneShot: %v", err)
		}
		return res
	}

	// Valid config → exit 0.
	if err := os.WriteFile(cfg, []byte(`{"version":3}`), 0644); err != nil {
		t.Fatal(err)
	}
	good := run()
	if good.ExitCode != 0 {
		t.Errorf("valid config: exit=%d, logs=%q", good.ExitCode, good.Logs)
	}
	_ = d.RemoveContainer(good.ID)

	// Invalid config → non-zero exit, with output explaining why.
	if err := os.WriteFile(cfg, []byte(`{`), 0644); err != nil {
		t.Fatal(err)
	}
	bad := run()
	if bad.ExitCode == 0 {
		t.Errorf("invalid config: expected non-zero exit, got 0 (logs=%q)", bad.Logs)
	}
	if bad.Logs == "" {
		t.Error("invalid config: expected diagnostic output, got empty logs")
	}
	_ = d.RemoveContainer(bad.ID)
}
