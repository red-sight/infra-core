package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"infra/registrator/internal/gateway"
	"infra/registrator/internal/registrar"
	"infra/registrator/internal/versiongate"
)

const (
	// managedConfigLabel marks Swarm config objects created by the registrator.
	managedConfigLabel = "infra.managed"
	// manifestLabel stores the per-service {version,hash} ledger as JSON on the
	// live config object — the cluster is its own ledger, no external store.
	manifestLabel = "infra.manifest"
	// configNamePrefix + render hash names each immutable config object.
	configNamePrefix = "krakend-config-"
)

// configRenderer renders the KrakenD config bytes and the per-service digests
// used by the version gate. Satisfied by *gateway.Generator.
type configRenderer interface {
	RenderWithDigests([]gateway.ServiceInfo, map[string][]string) ([]byte, map[string]versiongate.Digest, error)
}

// swarmDeliverer is the subset of DockerClient the artifact path needs. Kept as
// an interface so deliverArtifact is unit-testable with a fake.
type swarmDeliverer interface {
	Configs(label string) ([]registrar.ConfigSummary, error)
	CreateConfig(name string, data []byte, labels map[string]string) (string, error)
	UpdateServiceConfig(serviceName, target, configID, configName string) error
	ServiceConfigNames(serviceName string) ([]string, error)
}

// deliverArtifact renders the config, enforces the per-service version gate
// against the last-delivered manifest, and — if the render changed — publishes it
// as a new immutable Swarm config object and rolls the KrakenD service onto it.
//
// It never restarts anything directly: UpdateServiceConfig triggers Swarm's own
// rolling update. KrakenD config validity is expected to be checked in CI
// (`registrator -dry-run | krakend check`) before this runs; this function
// enforces the version policy and idempotent delivery.
func deliverArtifact(r configRenderer, d swarmDeliverer, services []gateway.ServiceInfo, scopeRoles map[string][]string, krakendService, configTarget string) error {
	data, digests, err := r.RenderWithDigests(services, scopeRoles)
	if err != nil {
		return fmt.Errorf("render: %w", err)
	}

	last, err := readManifest(d, krakendService)
	if err != nil {
		return fmt.Errorf("read current manifest: %w", err)
	}
	if violations := versiongate.Check(digests, last); len(violations) > 0 {
		return fmt.Errorf("version gate rejected delivery: %s", formatViolations(violations))
	}

	name := configNamePrefix + renderHash(data)

	managed, err := d.Configs(managedConfigLabel + "=true")
	if err != nil {
		return fmt.Errorf("list configs: %w", err)
	}
	for _, c := range managed {
		if c.Spec.Name == name {
			log.Printf("artifact: render unchanged (%s already delivered); nothing to do", name)
			return nil
		}
	}

	manifest, err := json.Marshal(digests)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	id, err := d.CreateConfig(name, data, map[string]string{
		managedConfigLabel: "true",
		manifestLabel:      string(manifest),
	})
	if err != nil {
		return fmt.Errorf("create config %s: %w", name, err)
	}

	if err := d.UpdateServiceConfig(krakendService, configTarget, id, name); err != nil {
		return fmt.Errorf("roll %s onto %s: %w", krakendService, name, err)
	}

	log.Printf("artifact: delivered %s to %s (%d service(s); %d managed config object(s) now exist)",
		name, krakendService, len(digests), len(managed)+1)
	return nil
}

// readManifest returns the {service: digest} ledger from the config object the
// KrakenD service currently mounts. Returns nil (no gate) on first delivery.
func readManifest(d swarmDeliverer, krakendService string) (map[string]versiongate.Digest, error) {
	names, err := d.ServiceConfigNames(krakendService)
	if err != nil {
		return nil, err
	}
	var current string
	for _, n := range names {
		if strings.HasPrefix(n, configNamePrefix) {
			current = n
			break
		}
	}
	if current == "" {
		return nil, nil // no managed config yet — first delivery
	}

	managed, err := d.Configs(managedConfigLabel + "=true")
	if err != nil {
		return nil, err
	}
	for _, c := range managed {
		if c.Spec.Name != current {
			continue
		}
		raw := c.Spec.Labels[manifestLabel]
		if raw == "" {
			return nil, nil
		}
		var manifest map[string]versiongate.Digest
		if err := json.Unmarshal([]byte(raw), &manifest); err != nil {
			return nil, fmt.Errorf("parse manifest on %s: %w", current, err)
		}
		return manifest, nil
	}
	return nil, nil
}

func renderHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

func formatViolations(vs []versiongate.Violation) string {
	parts := make([]string, len(vs))
	for i, v := range vs {
		parts[i] = fmt.Sprintf("%s changed its contract but info.version did not increase (%s → %s); bump it",
			v.Service, v.OldVersion, v.NewVersion)
	}
	return strings.Join(parts, "; ")
}
