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

// passRunner performs one full generate+deliver pass. The modes diverge before
// delivery (auto writes the local config and restarts the container; artifact
// renders and delivers a Swarm config object without writing local files), so
// each is its own runner rather than a shared generate()-then-deliver() split.
type passRunner interface {
	run()
}

// autoRunner (dev): regenerate the local files and restart the local KrakenD
// container so it re-reads the config. Restart is cheap with no prod traffic.
type autoRunner struct {
	registry *registrar.Registry
	agg      *openapi.Aggregator
	gen      *gateway.Generator
	lc       *logto.Client
	docker   *registrar.DockerClient
}

func (a *autoRunner) run() {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	res := generate(a.registry, a.agg, a.gen, a.lc)
	if !res.changed() {
		return
	}
	log.Printf("reload: %d service(s), openapi=%v krakend=%v", res.serviceCount, res.openapiChanged, res.krakendChanged)
	if err := a.docker.RestartContainerByLabel("com.docker.compose.service=krakend"); err != nil {
		log.Printf("krakend restart: %v", err)
	} else {
		log.Println("krakend restarting")
	}
}

// artifactRunner (prod): render the config, enforce the per-service version gate,
// and deliver it as an immutable Swarm config object, rolling the KrakenD service
// onto it. Never writes local files or restarts a container autonomously.
type artifactRunner struct {
	registry       *registrar.Registry
	gen            *gateway.Generator
	lc             *logto.Client
	docker         *registrar.DockerClient
	krakendService string
	configTarget   string
}

func (a *artifactRunner) run() {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	scopeRoles, err := a.lc.ScopeRoles()
	if err != nil {
		log.Printf("logto: scope→roles unavailable (%v); endpoints with required scopes will deny all until the mapping is available (fail-closed)", err)
	}
	if err := deliverArtifact(a.gen, a.docker, gatewayServices(a.registry), scopeRoles, a.krakendService, a.configTarget); err != nil {
		log.Printf("artifact delivery: %v", err)
	}
}

// genResult reports what a single generation pass produced.
type genResult struct {
	serviceCount   int
	openapiChanged bool
	krakendChanged bool
}

func (r genResult) changed() bool { return r.openapiChanged || r.krakendChanged }

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

// generate (auto mode) rebuilds the aggregated OpenAPI spec and the KrakenD
// config, writing both atomically as a side effect, and reports what changed.
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

// reloadMu serializes a runner's pass across its callers — the debounce timer and
// the poll ticker — so they cannot generate/deliver concurrently.
var reloadMu sync.Mutex
