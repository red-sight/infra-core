package registrar

type EnvironmentAdapter interface {
	WatchServices() <-chan ServiceEvent
	Mode() string // "compose" | "swarm"
}

type ServiceEvent struct {
	Name    string
	Port    int
	Healthy bool
	Removed bool
}
