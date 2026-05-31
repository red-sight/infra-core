package registrar

import "sync"

// EnvironmentAdapter is the source of ServiceEvents, abstracting Compose from Swarm.
type EnvironmentAdapter interface {
	WatchServices() <-chan ServiceEvent
	Mode() string // "compose" | "swarm"
}

// ServiceEvent is emitted when a service's health changes.
type ServiceEvent struct {
	Name          string
	Port          int
	OpenAPIRoute  string
	AuthProtected bool
	Healthy       bool
	Removed       bool
}

// Service is a healthy registered microservice.
type Service struct {
	Name          string
	Port          int
	OpenAPIRoute  string
	AuthProtected bool
}

// Registry maintains the current set of healthy services, safe for concurrent use.
type Registry struct {
	mu       sync.RWMutex
	services map[string]Service
}

func NewRegistry() *Registry {
	return &Registry{services: make(map[string]Service)}
}

func (r *Registry) Add(svc Service) {
	r.mu.Lock()
	r.services[svc.Name] = svc
	r.mu.Unlock()
}

func (r *Registry) Remove(name string) {
	r.mu.Lock()
	delete(r.services, name)
	r.mu.Unlock()
}

func (r *Registry) Services() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Service, 0, len(r.services))
	for _, svc := range r.services {
		out = append(out, svc)
	}
	return out
}
