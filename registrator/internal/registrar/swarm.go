package registrar

import (
	"context"
	"log"
)

// SwarmAdapter watches Docker service events for Swarm deployments.
// On any service change it re-scans all labeled services and emits events.
type SwarmAdapter struct {
	docker *DockerClient
	events chan ServiceEvent
	cancel context.CancelFunc
}

func NewSwarmAdapter(docker *DockerClient) *SwarmAdapter {
	return &SwarmAdapter{
		docker: docker,
		events: make(chan ServiceEvent, 32),
	}
}

func (a *SwarmAdapter) Mode() string { return "swarm" }

func (a *SwarmAdapter) WatchServices() <-chan ServiceEvent {
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	go a.run(ctx)
	return a.events
}

func (a *SwarmAdapter) run(ctx context.Context) {
	defer close(a.events)

	// Initial scan.
	a.scanServices(ctx)

	dockerEvents, errc := a.docker.Events(ctx, map[string][]string{
		"type":  {"service"},
		"label": {labelEnabled + "=true"},
	})

	for {
		select {
		case ev, ok := <-dockerEvents:
			if !ok {
				return
			}
			switch ev.Action {
			case "create", "update":
				a.scanServices(ctx)
			case "remove":
				if name := ev.Actor.Attributes[labelName]; name != "" {
					select {
					case a.events <- ServiceEvent{Name: name, Removed: true}:
					case <-ctx.Done():
						return
					}
				}
			}
		case err := <-errc:
			if err != nil {
				log.Printf("registrar(swarm): event stream error: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (a *SwarmAdapter) scanServices(ctx context.Context) {
	services, err := a.docker.SwarmServices(labelEnabled + "=true")
	if err != nil {
		log.Printf("registrar(swarm): list services: %v", err)
		return
	}
	for _, s := range services {
		svc, ok := serviceFromLabels(s.Spec.Labels)
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
