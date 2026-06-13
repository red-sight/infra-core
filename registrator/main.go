package main

import (
	"flag"
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
	once := flag.Bool("once", false, "run a single generate+deliver pass and exit (for CI after deploy convergence)")
	dryRun := flag.Bool("dry-run", false, "render the KrakenD config to stdout and exit; do not write, validate, or apply")
	flag.Parse()

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

	// Validate every generated config with `krakend check` before it is promoted,
	// unless explicitly disabled. Runs a one-shot KrakenD container; the image
	// should be pinned to match the running KrakenD in prod.
	if env("INFRA_REGISTRATOR_VALIDATE", "true") != "false" {
		v := krakendValidator{docker: docker, image: env("INFRA_KRAKEND_IMAGE", "krakend:latest")}
		gen.SetValidator(v.validate)
	}

	logtoClient := logto.New()

	rmode := resolveReloadMode(mode)
	log.Printf("reload mode: %s", rmode)
	var rl reloader
	switch rmode {
	case modeArtifact:
		rl = artifactReloader{}
	default:
		rl = autoReloader{docker: docker}
	}

	// One-shot modes: a single synchronous scan, then act and exit. No watcher,
	// no poll loop, no HTTP server.
	if *dryRun {
		n := scanOnce(adapter, registry)
		log.Printf("dry-run: %d service(s)", n)
		scopeRoles, err := logtoClient.ScopeRoles()
		if err != nil {
			log.Printf("logto: scope→roles unavailable (%v); rendering with deny-all for scoped endpoints (fail-closed)", err)
		}
		data, err := gen.Render(gatewayServices(registry), scopeRoles)
		if err != nil {
			log.Fatalf("dry-run render: %v", err)
		}
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))
		return
	}
	if *once {
		n := scanOnce(adapter, registry)
		log.Printf("once: %d service(s)", n)
		reload(registry, agg, gen, logtoClient, rl)
		return
	}

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
			reload(registry, agg, gen, logtoClient, rl)
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
				reload(registry, agg, gen, logtoClient, rl)
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

func parseDelay(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("invalid duration %q, defaulting to 5s", s)
		return 5 * time.Second
	}
	return d
}
