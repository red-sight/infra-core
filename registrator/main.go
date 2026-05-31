package main

import (
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"infra/registrator/internal/gateway"
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
		adapter = registrar.NewComposeAdapter(docker)
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

	delay := parseDelay(env("INFRA_REGISTRATOR_RELOAD_DELAY", "5s"))

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
		timer = time.AfterFunc(delay, func() {
			reload(registry, agg, gen, docker)
		})
	}

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

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	http.Handle("/openapi.json", agg.Handler())

	log.Println("listening on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func reload(registry *registrar.Registry, agg *openapi.Aggregator, gen *gateway.Generator, docker *registrar.DockerClient) {
	services := registry.Services()
	log.Printf("reload triggered: %d service(s)", len(services))

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

	if err := agg.Aggregate(opServices); err != nil {
		log.Printf("openapi aggregate: %v", err)
	} else {
		log.Println("openapi spec updated")
	}

	if err := gen.Generate(gwServices); err != nil {
		log.Printf("krakend config: %v", err)
		return
	}
	log.Println("krakend config updated")

	krakendLabel := "com.docker.compose.service=krakend"
	if err := docker.RestartContainerByLabel(krakendLabel); err != nil {
		log.Printf("krakend restart: %v", err)
	} else {
		log.Println("krakend restarting")
	}
}

func parseDelay(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("invalid reload delay %q, defaulting to 5s", s)
		return 5 * time.Second
	}
	return d
}
