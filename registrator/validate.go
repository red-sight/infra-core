package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"infra/registrator/internal/registrar"
)

const krakendCheckContainer = "krakend-check"

// krakendValidator validates a candidate KrakenD config by running `krakend check`
// in a one-shot container named krakend-check. The candidate lives in the config
// directory bind-mounted into the registrator; the check container reuses that same
// bind, so it sees the candidate at the same path. The container is removed on
// success and left for inspection on failure — its name makes its purpose obvious.
type krakendValidator struct {
	docker *registrar.DockerClient
	image  string
}

func (v krakendValidator) validate(candidatePath string) error {
	dest := filepath.Dir(candidatePath)
	src, err := v.docker.SelfBindSource(dest)
	if err != nil {
		return fmt.Errorf("locate config mount: %w", err)
	}

	res, err := v.docker.RunOneShot(registrar.OneShotSpec{
		Name:        krakendCheckContainer,
		Image:       v.image,
		Cmd:         []string{"check", "-c", candidatePath},
		Binds:       []string{src + ":" + dest + ":ro"},
		NetworkNone: true,
	})
	if err != nil {
		return fmt.Errorf("run %s: %w", krakendCheckContainer, err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("krakend check failed (exit %d): %s", res.ExitCode, strings.TrimSpace(res.Logs))
	}

	_ = v.docker.RemoveContainer(res.ID) // best-effort cleanup on success
	return nil
}
