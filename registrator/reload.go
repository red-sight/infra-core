package main

import (
	"log"
	"sync"

	"infra/registrator/internal/gateway"
	"infra/registrator/internal/logto"
	"infra/registrator/internal/openapi"
	"infra/registrator/internal/registrar"
)

// reloadMode selects what happens to a freshly generated config — the only thing
// that differs between environments. Generation itself is identical.
type reloadMode string

const (
	modeAuto     reloadMode = "auto"     // dev: write config + restart the local KrakenD container
	modeArtifact reloadMode = "artifact" // prod: regenerate only, never restart autonomously
)

// resolveReloadMode reads INFRA_REGISTRATOR_RELOAD_MODE, defaulting to a value
// inferred from the Docker mode: compose→auto, swarm→artifact.
func resolveReloadMode(dockerMode string) reloadMode {
	switch v := env("INFRA_REGISTRATOR_RELOAD_MODE", ""); v {
	case string(modeAuto):
		return modeAuto
	case string(modeArtifact):
		return modeArtifact
	case "":
		if dockerMode == "swarm" {
			return modeArtifact
		}
		return modeAuto
	default:
		log.Fatalf("invalid INFRA_REGISTRATOR_RELOAD_MODE %q (want auto|artifact)", v)
		return ""
	}
}

// reloader performs the "last mile" after a config regeneration. Both modes share
// the same generation; only delivery differs.
type reloader interface {
	deliver(res genResult) error
}

// autoReloader restarts the local KrakenD container so it re-reads the freshly
// written config. For local dev only: restart is cheap with no production traffic.
type autoReloader struct {
	docker *registrar.DockerClient
}

func (a autoReloader) deliver(genResult) error {
	if err := a.docker.RestartContainerByLabel("com.docker.compose.service=krakend"); err != nil {
		return err
	}
	log.Println("krakend restarting")
	return nil
}

// artifactReloader is the production last mile: the config is regenerated and
// written, but KrakenD is never restarted autonomously — applying it is a
// separate, gated deploy step (no API appears on the fly). Artifact emission —
// immutable config object, contract-hash version gate — lands in a later phase;
// for now this only enforces the "no autonomous reload" guarantee.
type artifactReloader struct{}

func (artifactReloader) deliver(res genResult) error {
	log.Printf("artifact mode: config regenerated (%d service(s)); not restarting KrakenD — apply is a gated deploy step", res.serviceCount)
	return nil
}

// genResult reports what a single generation pass produced.
type genResult struct {
	serviceCount   int
	openapiChanged bool
	krakendChanged bool
}

func (r genResult) changed() bool { return r.openapiChanged || r.krakendChanged }

// generate snapshots the registry, rebuilds the aggregated OpenAPI spec and the
// KrakenD config (writing both atomically as a side effect), and reports what
// changed. It never restarts anything — that is the reloader's responsibility.
// scanOnce performs a single synchronous discovery pass and populates the registry
// with the eligible (healthy, non-removed) services it finds. Returns the count.
func scanOnce(adapter registrar.EnvironmentAdapter, registry *registrar.Registry) int {
	for _, ev := range adapter.Scan() {
		if ev.Healthy && !ev.Removed {
			registry.Add(registrar.Service{
				Name:          ev.Name,
				Port:          ev.Port,
				OpenAPIRoute:  ev.OpenAPIRoute,
				AuthProtected: ev.AuthProtected,
			})
		}
	}
	return len(registry.Services())
}

func gatewayServices(registry *registrar.Registry) []gateway.ServiceInfo {
	services := registry.Services()
	out := make([]gateway.ServiceInfo, len(services))
	for i, svc := range services {
		out[i] = gateway.ServiceInfo{
			Name:          svc.Name,
			Port:          svc.Port,
			OpenAPIRoute:  svc.OpenAPIRoute,
			AuthProtected: svc.AuthProtected,
		}
	}
	return out
}

func openapiServices(registry *registrar.Registry) []openapi.ServiceInfo {
	services := registry.Services()
	out := make([]openapi.ServiceInfo, len(services))
	for i, svc := range services {
		out[i] = openapi.ServiceInfo{
			Name:          svc.Name,
			Port:          svc.Port,
			OpenAPIRoute:  svc.OpenAPIRoute,
			AuthProtected: svc.AuthProtected,
		}
	}
	return out
}

func generate(registry *registrar.Registry, agg *openapi.Aggregator, gen *gateway.Generator, lc *logto.Client) genResult {
	res := genResult{serviceCount: len(registry.Services())}

	opChanged, err := agg.Aggregate(openapiServices(registry))
	if err != nil {
		log.Printf("openapi aggregate: %v", err)
	}
	res.openapiChanged = opChanged

	scopeRoles, err := lc.ScopeRoles()
	if err != nil {
		log.Printf("logto: scope→roles unavailable (%v); endpoints with required scopes will deny all until the mapping is available (fail-closed)", err)
	}

	gwChanged, err := gen.Generate(gatewayServices(registry), scopeRoles)
	if err != nil {
		log.Printf("krakend config: %v", err)
		return res
	}
	res.krakendChanged = gwChanged

	return res
}

// reloadMu serializes reload across its callers — the debounce timer and the poll
// ticker — so they cannot generate and write the config concurrently.
var reloadMu sync.Mutex

// reload regenerates the config and, if anything changed, hands delivery to the reloader.
func reload(registry *registrar.Registry, agg *openapi.Aggregator, gen *gateway.Generator, lc *logto.Client, r reloader) {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	res := generate(registry, agg, gen, lc)
	if !res.changed() {
		return // nothing changed, skip delivery
	}

	log.Printf("reload: %d service(s), openapi=%v krakend=%v", res.serviceCount, res.openapiChanged, res.krakendChanged)
	if err := r.deliver(res); err != nil {
		log.Printf("deliver: %v", err)
	}
}
