package registrar

import (
	"context"
	"log"
	"strconv"
	"time"
)

const (
	labelEnabled      = "infra.enabled"
	labelName         = "infra.name"
	labelPort         = "infra.port"
	labelOpenAPIRoute = "infra.openapi-route"
	labelAuthProtected = "infra.auth.protected"
)

// ComposeAdapter watches Docker container health events for Compose deployments.
type ComposeAdapter struct {
	docker          *DockerClient
	events          chan ServiceEvent
	cancel          context.CancelFunc
	reconcileEvery  time.Duration
}

func NewComposeAdapter(docker *DockerClient, reconcileEvery time.Duration) *ComposeAdapter {
	return &ComposeAdapter{
		docker:         docker,
		events:         make(chan ServiceEvent, 32),
		reconcileEvery: reconcileEvery,
	}
}

func (a *ComposeAdapter) Mode() string { return "compose" }

func (a *ComposeAdapter) WatchServices() <-chan ServiceEvent {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go a.run(ctx)
	return a.events
}

func (a *ComposeAdapter) run(ctx context.Context) {
	defer close(a.events)

	// Emit events for containers already healthy at startup.
	a.scanHealthy(ctx)

	dockerEvents, errc := a.docker.Events(ctx, map[string][]string{
		"type":  {"container"},
		"label": {labelEnabled + "=true"},
	})

	reconcile := time.NewTicker(a.reconcileEvery)
	defer reconcile.Stop()

	for {
		select {
		case ev, ok := <-dockerEvents:
			if !ok {
				return
			}
			a.handleEvent(ctx, ev)
		case err := <-errc:
			if err != nil {
				log.Printf("registrar(compose): event stream error: %v", err)
			}
		case <-reconcile.C:
			a.scanHealthy(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (a *ComposeAdapter) scanHealthy(ctx context.Context) {
	containers, err := a.docker.Containers(labelEnabled + "=true")
	if err != nil {
		log.Printf("registrar(compose): initial scan: %v", err)
		return
	}
	for _, c := range containers {
		detail, err := a.docker.ContainerInspect(c.ID)
		if err != nil {
			log.Printf("registrar(compose): inspect %s: %v", c.ID[:12], err)
			continue
		}
		if detail.State.Health.Status != "healthy" {
			continue
		}
		svc, ok := serviceFromLabels(detail.Config.Labels)
		if !ok {
			continue
		}
		select {
		case a.events <- toEvent(svc, true, false):
		case <-ctx.Done():
			return
		}
	}
}

func (a *ComposeAdapter) handleEvent(ctx context.Context, ev DockerEvent) {
	switch ev.Action {
	case "health_status":
		if ev.Actor.Attributes["health_status"] == "healthy" {
			detail, err := a.docker.ContainerInspect(ev.Actor.ID)
			if err != nil {
				log.Printf("registrar(compose): inspect %s: %v", ev.Actor.ID[:12], err)
				return
			}
			if svc, ok := serviceFromLabels(detail.Config.Labels); ok {
				select {
				case a.events <- toEvent(svc, true, false):
				case <-ctx.Done():
				}
			}
		} else {
			// unhealthy — pull the service from the registry
			if name := ev.Actor.Attributes[labelName]; name != "" {
				select {
				case a.events <- ServiceEvent{Name: name, Healthy: false}:
				case <-ctx.Done():
				}
			}
		}

	case "die", "stop", "kill":
		if name := ev.Actor.Attributes[labelName]; name != "" {
			select {
			case a.events <- ServiceEvent{Name: name, Removed: true}:
			case <-ctx.Done():
			}
		}
	}
}

// serviceFromLabels builds a Service from Docker container labels.
// Returns false if the container is not a valid infra service.
func serviceFromLabels(labels map[string]string) (Service, bool) {
	if labels[labelEnabled] != "true" {
		return Service{}, false
	}
	name := labels[labelName]
	if name == "" {
		return Service{}, false
	}

	port, _ := strconv.Atoi(labels[labelPort])
	if port == 0 {
		port = 3000
	}

	route := labels[labelOpenAPIRoute]
	if route == "" {
		route = "openapi"
	}

	protected := true
	if v, ok := labels[labelAuthProtected]; ok {
		protected = v != "false"
	}

	return Service{
		Name:          name,
		Port:          port,
		OpenAPIRoute:  route,
		AuthProtected: protected,
	}, true
}

func toEvent(svc Service, healthy, removed bool) ServiceEvent {
	return ServiceEvent{
		Name:          svc.Name,
		Port:          svc.Port,
		OpenAPIRoute:  svc.OpenAPIRoute,
		AuthProtected: svc.AuthProtected,
		Healthy:       healthy,
		Removed:       removed,
	}
}
