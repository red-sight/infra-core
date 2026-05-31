package registrar

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const dockerBase = "http://localhost/v1.47"

// DockerClient talks to the Docker daemon via the Unix socket.
type DockerClient struct {
	http *http.Client
}

func NewDockerClient() *DockerClient {
	return &DockerClient{
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, "unix", "/var/run/docker.sock")
				},
			},
		},
	}
}

func (d *DockerClient) get(path string, query url.Values) (*http.Response, error) {
	u := dockerBase + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	return d.http.Get(u) //nolint:gosec
}

func (d *DockerClient) post(path string) (*http.Response, error) {
	return d.http.Post(dockerBase+path, "application/json", nil) //nolint:gosec
}

// --- Mode detection ---

type dockerInfo struct {
	Swarm struct {
		LocalNodeState string `json:"LocalNodeState"`
	} `json:"Swarm"`
}

// DetectMode returns "swarm" if this node is a Swarm member, otherwise "compose".
func (d *DockerClient) DetectMode() (string, error) {
	resp, err := d.get("/info", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var info dockerInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", err
	}
	if info.Swarm.LocalNodeState == "active" {
		return "swarm", nil
	}
	return "compose", nil
}

// --- Container API ---

// ContainerSummary is the minimal shape returned by GET /containers/json.
type ContainerSummary struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Labels map[string]string `json:"Labels"`
	State  string            `json:"State"`
}

// ContainerDetail is returned by GET /containers/{id}/json.
type ContainerDetail struct {
	ID   string `json:"Id"`
	Name string `json:"Name"`
	Config struct {
		Labels       map[string]string      `json:"Labels"`
		ExposedPorts map[string]interface{} `json:"ExposedPorts"`
	} `json:"Config"`
	State struct {
		Health struct {
			Status string `json:"Status"` // "healthy", "unhealthy", "starting"
		} `json:"Health"`
	} `json:"State"`
}

// Containers returns running containers matching label (e.g. "infra.enabled=true").
func (d *DockerClient) Containers(label string) ([]ContainerSummary, error) {
	filters := fmt.Sprintf(`{"label":[%q]}`, label)
	resp, err := d.get("/containers/json", url.Values{"filters": {filters}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []ContainerSummary
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// ContainerInspect returns full details for a container by ID or name.
func (d *DockerClient) ContainerInspect(id string) (*ContainerDetail, error) {
	resp, err := d.get("/containers/"+id+"/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out ContainerDetail
	return &out, json.NewDecoder(resp.Body).Decode(&out)
}

// RestartContainerByLabel finds the first container matching a label and restarts it.
func (d *DockerClient) RestartContainerByLabel(label string) error {
	containers, err := d.Containers(label)
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}
	if len(containers) == 0 {
		return fmt.Errorf("no container with label %q", label)
	}
	id := containers[0].ID
	resp, err := d.post("/containers/" + id + "/restart")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("restart HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// --- Swarm service API ---

// SwarmService is the minimal shape of a Swarm service from GET /services.
type SwarmService struct {
	ID   string `json:"ID"`
	Spec struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Spec"`
}

// SwarmServices returns Swarm services matching the given label filter.
func (d *DockerClient) SwarmServices(label string) ([]SwarmService, error) {
	filters := fmt.Sprintf(`{"label":[%q]}`, label)
	resp, err := d.get("/services", url.Values{"filters": {filters}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []SwarmService
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// --- Event streaming ---

// DockerEvent is the shape of a single event from GET /events.
type DockerEvent struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
}

// Events streams Docker events matching the given filters until ctx is cancelled.
// Reconnects automatically on transient errors.
func (d *DockerClient) Events(ctx context.Context, filters map[string][]string) (<-chan DockerEvent, <-chan error) {
	events := make(chan DockerEvent, 32)
	errc := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errc)
		for {
			if err := d.streamEvents(ctx, filters, events); err != nil {
				if ctx.Err() != nil {
					return
				}
				select {
				case errc <- err:
				default:
				}
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return
				}
			} else if ctx.Err() != nil {
				return
			}
		}
	}()

	return events, errc
}

func (d *DockerClient) streamEvents(ctx context.Context, filters map[string][]string, out chan<- DockerEvent) error {
	filterJSON, err := json.Marshal(filters)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", dockerBase+"/events", nil)
	if err != nil {
		return err
	}
	q := req.URL.Query()
	q.Set("filters", string(filterJSON))
	req.URL.RawQuery = q.Encode()

	resp, err := d.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev DockerEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		select {
		case out <- ev:
		case <-ctx.Done():
			return nil
		}
	}
	return scanner.Err()
}
