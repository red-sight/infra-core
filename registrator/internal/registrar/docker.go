package registrar

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
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

func (d *DockerClient) postJSON(path string, body interface{}) (*http.Response, error) {
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	}
	return d.http.Post(dockerBase+path, "application/json", buf) //nolint:gosec
}

func (d *DockerClient) httpDelete(path string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", dockerBase+path, nil)
	if err != nil {
		return nil, err
	}
	return d.http.Do(req)
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
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Labels       map[string]string      `json:"Labels"`
		ExposedPorts map[string]interface{} `json:"ExposedPorts"`
	} `json:"Config"`
	State struct {
		Health struct {
			Status string `json:"Status"` // "healthy", "unhealthy", "starting"
		} `json:"Health"`
	} `json:"State"`
	Mounts []struct {
		Type        string `json:"Type"`   // "bind" or "volume"
		Name        string `json:"Name"`   // volume name (empty for binds)
		Source      string `json:"Source"` // host path (bind) or volume mountpoint
		Destination string `json:"Destination"`
	} `json:"Mounts"`
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

// --- One-shot containers (used for `krakend check` validation) ---

// OneShotSpec describes a short-lived container to create, run to completion, and
// inspect. It is not auto-removed: the caller removes it on success and may leave
// it for inspection on failure (its Name makes its purpose self-evident).
type OneShotSpec struct {
	Name        string   // container name, e.g. "krakend-check"
	Image       string   // image reference
	Cmd         []string // command + args
	Binds       []string // HostConfig.Binds, e.g. "/host/path:/etc/krakend:ro"
	NetworkNone bool     // run with no network (NetworkMode "none")
}

// OneShotResult is the outcome of a finished one-shot container.
type OneShotResult struct {
	ID       string
	ExitCode int
	Logs     string // combined stdout+stderr
}

// RunOneShot creates a container from spec, starts it, waits for it to exit, and
// returns its exit code and logs. Any pre-existing container with the same name
// (e.g. a previous failed run left for inspection) is removed first. The created
// container is left in place; the caller decides whether to remove it.
func (d *DockerClient) RunOneShot(spec OneShotSpec) (OneShotResult, error) {
	var res OneShotResult

	if spec.Name != "" {
		_ = d.RemoveContainer(spec.Name) // best-effort: clear a stale run
	}

	hostConfig := map[string]interface{}{"Binds": spec.Binds}
	if spec.NetworkNone {
		hostConfig["NetworkMode"] = "none"
	}
	createBody := map[string]interface{}{
		"Image":      spec.Image,
		"Cmd":        spec.Cmd,
		"Tty":        true, // raw (non-multiplexed) log stream
		"HostConfig": hostConfig,
	}

	path := "/containers/create"
	if spec.Name != "" {
		path += "?" + url.Values{"name": {spec.Name}}.Encode()
	}
	resp, err := d.postJSON(path, createBody)
	if err != nil {
		return res, fmt.Errorf("create: %w", err)
	}
	created, err := decodeOrError(resp, "create", struct {
		ID string `json:"Id"`
	}{})
	if err != nil {
		return res, err
	}
	res.ID = created.ID

	startResp, err := d.post("/containers/" + res.ID + "/start")
	if err != nil {
		return res, fmt.Errorf("start: %w", err)
	}
	if err := expectStatus(startResp, "start"); err != nil {
		return res, err
	}

	waitResp, err := d.post("/containers/" + res.ID + "/wait")
	if err != nil {
		return res, fmt.Errorf("wait: %w", err)
	}
	wait, err := decodeOrError(waitResp, "wait", struct {
		StatusCode int `json:"StatusCode"`
	}{})
	if err != nil {
		return res, err
	}
	res.ExitCode = wait.StatusCode

	logsResp, err := d.get("/containers/"+res.ID+"/logs", url.Values{"stdout": {"1"}, "stderr": {"1"}})
	if err != nil {
		return res, fmt.Errorf("logs: %w", err)
	}
	defer logsResp.Body.Close()
	logBytes, _ := io.ReadAll(logsResp.Body)
	res.Logs = string(logBytes)

	return res, nil
}

// RemoveContainer force-removes a container by ID or name. A missing container
// (404) is not an error — the caller often removes pre-emptively.
func (d *DockerClient) RemoveContainer(id string) error {
	resp, err := d.httpDelete("/containers/" + id + "?force=1")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// SelfBindSource inspects this registrator container and returns the host source
// of the mount whose destination is destDir. It lets a sibling container reuse the
// same bind (e.g. the KrakenD config dir) without knowing the host layout.
func (d *DockerClient) SelfBindSource(destDir string) (string, error) {
	id, err := os.Hostname() // Docker sets the container hostname to its short ID
	if err != nil {
		return "", err
	}
	detail, err := d.ContainerInspect(id)
	if err != nil {
		return "", fmt.Errorf("inspect self (%s): %w", id, err)
	}
	for _, m := range detail.Mounts {
		if m.Destination == destDir {
			return m.Source, nil
		}
	}
	return "", fmt.Errorf("no mount with destination %q on self (%s)", destDir, id)
}

// expectStatus closes resp and returns an error if its status is not 2xx.
func expectStatus(resp *http.Response, op string) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s HTTP %d: %s", op, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// decodeOrError closes resp; on a 2xx it decodes the body into a value shaped like
// out and returns it, otherwise it returns the HTTP error.
func decodeOrError[T any](resp *http.Response, op string, out T) (T, error) {
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return out, fmt.Errorf("%s HTTP %d: %s", op, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("%s decode: %w", op, err)
	}
	return out, nil
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

// --- Swarm config objects & config delivery ---

// ConfigSummary is the minimal shape of a Swarm config from GET /configs.
type ConfigSummary struct {
	ID   string `json:"ID"`
	Spec struct {
		Name   string            `json:"Name"`
		Labels map[string]string `json:"Labels"`
	} `json:"Spec"`
}

// CreateConfig creates an immutable Swarm config object and returns its ID.
// Config objects are content-addressed by the caller (name = krakend-config-<hash>),
// so a 409 name conflict means "identical config already delivered".
func (d *DockerClient) CreateConfig(name string, data []byte, labels map[string]string) (string, error) {
	body := map[string]interface{}{
		"Name":   name,
		"Labels": labels,
		"Data":   base64.StdEncoding.EncodeToString(data),
	}
	resp, err := d.postJSON("/configs/create", body)
	if err != nil {
		return "", fmt.Errorf("create config: %w", err)
	}
	out, err := decodeOrError(resp, "create config", struct {
		ID string `json:"ID"`
	}{})
	if err != nil {
		return "", err
	}
	return out.ID, nil
}

// Configs returns Swarm config objects matching the given label filter (e.g.
// "infra.managed=true").
func (d *DockerClient) Configs(label string) ([]ConfigSummary, error) {
	filters := fmt.Sprintf(`{"label":[%q]}`, label)
	resp, err := d.get("/configs", url.Values{"filters": {filters}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []ConfigSummary
	return out, json.NewDecoder(resp.Body).Decode(&out)
}

// RemoveConfig deletes a Swarm config object by ID. A config still in use by a
// service cannot be removed (Docker returns an error) — remove it only after the
// service has been updated off it.
func (d *DockerClient) RemoveConfig(id string) error {
	resp, err := d.httpDelete("/configs/" + id)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("remove config HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

// swarmServiceDetail carries the parts of a service we need to update it: its ID,
// the version index required by the update API, and the full spec to resubmit.
type swarmServiceDetail struct {
	ID      string                 `json:"ID"`
	Version struct{ Index int }    `json:"Version"`
	Spec    map[string]interface{} `json:"Spec"`
}

// serviceByName returns the service whose Spec.Name is exactly name (Docker's name
// filter is a substring match, so we filter precisely). Returns nil if not found.
func (d *DockerClient) serviceByName(name string) (*swarmServiceDetail, error) {
	filters := fmt.Sprintf(`{"name":[%q]}`, name)
	resp, err := d.get("/services", url.Values{"filters": {filters}})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []swarmServiceDetail
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	for i := range out {
		if n, _ := out[i].Spec["Name"].(string); n == name {
			return &out[i], nil
		}
	}
	return nil, nil
}

// UpdateServiceConfig points a service's ContainerSpec at configName (mounted at
// targetPath) and bumps ForceUpdate, triggering a rolling restart. This is the
// Swarm equivalent of the auto-mode container restart: the service rolls onto the
// new config object with order:start-first (set in the stack).
func (d *DockerClient) UpdateServiceConfig(serviceName, targetPath, configID, configName string) error {
	svc, err := d.serviceByName(serviceName)
	if err != nil {
		return fmt.Errorf("inspect service %q: %w", serviceName, err)
	}
	if svc == nil {
		return fmt.Errorf("service %q not found", serviceName)
	}

	tt, ok := svc.Spec["TaskTemplate"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("service %q: no TaskTemplate", serviceName)
	}
	cs, ok := tt["ContainerSpec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("service %q: no ContainerSpec", serviceName)
	}

	cs["Configs"] = []interface{}{
		map[string]interface{}{
			"ConfigID":   configID,
			"ConfigName": configName,
			"File": map[string]interface{}{
				"Name": targetPath,
				"UID":  "0",
				"GID":  "0",
				"Mode": 0o444,
			},
		},
	}
	tt["ForceUpdate"] = forceUpdateNext(tt["ForceUpdate"])

	resp, err := d.postJSON(fmt.Sprintf("/services/%s/update?version=%d", svc.ID, svc.Version.Index), svc.Spec)
	if err != nil {
		return fmt.Errorf("update service %q: %w", serviceName, err)
	}
	return expectStatus(resp, "service update")
}

// ServiceConfigNames returns the config object names currently mounted in a
// service's ContainerSpec.
func (d *DockerClient) ServiceConfigNames(serviceName string) ([]string, error) {
	svc, err := d.serviceByName(serviceName)
	if err != nil || svc == nil {
		return nil, err
	}
	tt, _ := svc.Spec["TaskTemplate"].(map[string]interface{})
	cs, _ := tt["ContainerSpec"].(map[string]interface{})
	configs, _ := cs["Configs"].([]interface{})
	var names []string
	for _, c := range configs {
		if cm, ok := c.(map[string]interface{}); ok {
			if n, ok := cm["ConfigName"].(string); ok {
				names = append(names, n)
			}
		}
	}
	return names, nil
}

// forceUpdateNext increments the ForceUpdate counter (JSON numbers decode as float64).
func forceUpdateNext(v interface{}) int {
	if f, ok := v.(float64); ok {
		return int(f) + 1
	}
	return 1
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
