package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"infra/registrator/internal/gateway"
	"infra/registrator/internal/logto"
	"infra/registrator/internal/openapi"
	"infra/registrator/internal/registrar"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("[registrator] ")

	docker := registrar.NewDockerClient()

	mode := env("INFRA_MODE", "")
	if mode == "" {
		var err error
		mode, err = docker.DetectMode()
		if err != nil {
			log.Fatalf("auto-detect mode: %v", err)
		}
	}
	log.Printf("mode: %s", mode)

	var adapter registrar.EnvironmentAdapter
	switch mode {
	case "swarm":
		adapter = registrar.NewSwarmAdapter(docker)
	default:
		reconcileEvery := parseDelay(env("INFRA_REGISTRATOR_RECONCILE_INTERVAL", "30s"))
		adapter = registrar.NewComposeAdapter(docker, reconcileEvery)
	}

	registry := registrar.NewRegistry()

	agg := openapi.New(openapi.Config{
		HTTPProtocol:  env("INFRA_HTTP_PROTOCOL", "http"),
		BaseDomain:    env("INFRA_HTTP_BASE_DOMAIN", "app.localhost"),
		OIDCSubdomain: env("INFRA_HTTP_OIDC_SUBDOMAIN", "auth"),
		APIRoute:      env("INFRA_API_ROUTE", "api"),
		SpecsPath:     env("INFRA_SPECS_PATH", "/specs"),
	})

	gen := gateway.New(gateway.Config{
		HTTPProtocol:    env("INFRA_HTTP_PROTOCOL", "http"),
		BaseDomain:      env("INFRA_HTTP_BASE_DOMAIN", "app.localhost"),
		OIDCSubdomain:   env("INFRA_HTTP_OIDC_SUBDOMAIN", "auth"),
		APIRoute:        env("INFRA_API_ROUTE", "api"),
		LogtoResourceID: env("INFRA_LOGTO_API_RESOURCE_ID", ""),
		ConfigPath:      env("INFRA_KRAKEND_CONFIG_PATH", "/etc/krakend/krakend.json"),
	})

	logtoClient := logto.New()

	debounceDelay := parseDelay(env("INFRA_REGISTRATOR_RELOAD_DELAY", "5s"))
	pollInterval := parseDelay(env("INFRA_REGISTRATOR_POLL_INTERVAL", "30s"))

	var (
		timerMu sync.Mutex
		timer   *time.Timer
	)
	resetDebounce := func() {
		timerMu.Lock()
		defer timerMu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(debounceDelay, func() {
			reload(registry, agg, gen, logtoClient, docker)
		})
	}

	// React to Docker health events.
	go func() {
		for ev := range adapter.WatchServices() {
			switch {
			case ev.Removed:
				registry.Remove(ev.Name)
				log.Printf("unregistered: %s", ev.Name)
			case ev.Healthy:
				registry.Add(registrar.Service{
					Name:          ev.Name,
					Port:          ev.Port,
					OpenAPIRoute:  ev.OpenAPIRoute,
					AuthProtected: ev.AuthProtected,
				})
				log.Printf("registered: %s (port %d)", ev.Name, ev.Port)
			default:
				registry.Remove(ev.Name)
				log.Printf("unhealthy: %s", ev.Name)
			}
			resetDebounce()
		}
	}()

	// Periodically re-fetch specs to catch in-process restarts (watch/hot-reload mode).
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for range ticker.C {
			if len(registry.Services()) > 0 {
				reload(registry, agg, gen, logtoClient, docker)
			}
		}
	}()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/openapi.json", agg.Handler())

	log.Println("listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

// reloadMu serializes reload across its two callers — the debounce timer and the
// poll ticker — so they cannot generate and write the config concurrently.
var reloadMu sync.Mutex

func reload(registry *registrar.Registry, agg *openapi.Aggregator, gen *gateway.Generator, lc *logto.Client, docker *registrar.DockerClient) {
	reloadMu.Lock()
	defer reloadMu.Unlock()

	services := registry.Services()

	opServices := make([]openapi.ServiceInfo, len(services))
	gwServices := make([]gateway.ServiceInfo, len(services))
	for i, svc := range services {
		opServices[i] = openapi.ServiceInfo{
			Name:          svc.Name,
			Port:          svc.Port,
			OpenAPIRoute:  svc.OpenAPIRoute,
			AuthProtected: svc.AuthProtected,
		}
		gwServices[i] = gateway.ServiceInfo{
			Name:          svc.Name,
			Port:          svc.Port,
			OpenAPIRoute:  svc.OpenAPIRoute,
			AuthProtected: svc.AuthProtected,
		}
	}

	opChanged, err := agg.Aggregate(opServices)
	if err != nil {
		log.Printf("openapi aggregate: %v", err)
	}

	scopeRoles, err := lc.ScopeRoles()
	if err != nil {
		log.Printf("logto: scope→roles unavailable (%v); endpoints with required scopes will deny all until the mapping is available (fail-closed)", err)
	}

	gwChanged, err := gen.Generate(gwServices, scopeRoles)
	if err != nil {
		log.Printf("krakend config: %v", err)
		return
	}

	if !opChanged && !gwChanged {
		return // nothing changed, skip KrakenD restart
	}

	log.Printf("reload: %d service(s), openapi=%v krakend=%v", len(services), opChanged, gwChanged)

	if err := docker.RestartContainerByLabel("com.docker.compose.service=krakend"); err != nil {
		log.Printf("krakend restart: %v", err)
	} else {
		log.Println("krakend restarting")
	}
}

func parseDelay(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("invalid duration %q, defaulting to 5s", s)
		return 5 * time.Second
	}
	return d
}
