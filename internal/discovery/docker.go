package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/models"
)

type DockerDiscovery struct {
	client      *http.Client
	db          *database.DB
	labelPrefix string
	network     string
	interval    time.Duration
	socketPath  string
}

func NewDockerDiscovery(db *database.DB, socketPath, labelPrefix, network string, interval time.Duration) (*DockerDiscovery, error) {
	if !strings.HasPrefix(socketPath, "unix://") {
		socketPath = "unix://" + socketPath
	}

	d := &DockerDiscovery{
		db:          db,
		labelPrefix: labelPrefix,
		network:     network,
		interval:    interval,
		socketPath:  strings.TrimPrefix(socketPath, "unix://"),
	}

	d.client = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", d.socketPath, 5*time.Second)
			},
		},
	}

	return d, nil
}

func (d *DockerDiscovery) Start(ctx context.Context) {
	slog.Info("starting docker discovery", "socket", d.socketPath, "interval", d.interval)

	d.Scan(ctx)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.Scan(ctx)
		case <-ctx.Done():
			slog.Info("docker discovery stopped")
			return
		}
	}
}

type dockerContainer struct {
	ID              string            `json:"Id"`
	Names           []string          `json:"Names"`
	Image           string            `json:"Image"`
	State           string            `json:"State"`
	Status          string            `json:"Status"`
	Labels          map[string]string `json:"Labels"`
	Ports           []dockerPort      `json:"Ports"`
	NetworkSettings *struct {
		Networks map[string]dockerNetwork `json:"Networks"`
	} `json:"NetworkSettings,omitempty"`
}

type dockerPort struct {
	PrivatePort uint16 `json:"PrivatePort"`
	PublicPort  uint16 `json:"PublicPort"`
	Type        string `json:"Type"`
}

type dockerNetwork struct {
	IPAddress string `json:"IPAddress"`
}

func (d *DockerDiscovery) Scan(ctx context.Context) {
	filter := fmt.Sprintf(`{"label":{"%s.enable":true}}`, d.labelPrefix)
	apiURL := fmt.Sprintf("http://localhost/containers/json?filters=%s", url.QueryEscape(filter))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		slog.Warn("docker discovery: error creating request", "error", err)
		return
	}

	resp, err := d.client.Do(req)
	if err != nil {
		slog.Warn("docker discovery: connection error", "error", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		slog.Warn("docker discovery: API error", "status", resp.StatusCode, "body", string(body))
		return
	}

	var containers []dockerContainer
	if err := json.Unmarshal(body, &containers); err != nil {
		slog.Error("docker discovery: decode error", "body", string(body))
		return
	}

	for _, c := range containers {
		if c.State != "running" {
			continue
		}

		svc := d.containerToService(c)
		if svc == nil {
			continue
		}

		id, err := d.db.CreateService(svc)
		if err != nil {
			slog.Warn("docker discovery: save error", "service", svc.Name, "error", err)
			continue
		}
		slog.Info("docker discovery: service discovered", "service", svc.Name, "id", id)
	}

	if len(containers) == 0 {
		slog.Debug("docker discovery: no containers found", "label_prefix", d.labelPrefix)
	}
}

func (d *DockerDiscovery) containerToService(c dockerContainer) *models.Service {
	labels := c.Labels

	name := labels[d.labelPrefix+".service"]
	if name == "" {
		for _, n := range c.Names {
			name = strings.TrimPrefix(n, "/")
			break
		}
	}

	displayName := labels[d.labelPrefix+".name"]
	if displayName == "" {
		displayName = name
	}

	svc := &models.Service{
		Name:        name,
		DisplayName: displayName,
		Description: labels[d.labelPrefix+".description"],
		Category:    labels[d.labelPrefix+".category"],
		Group:       labels[d.labelPrefix+".group"],
		Icon:        labels[d.labelPrefix+".icon"],
		HealthPath:  labels[d.labelPrefix+".health"],
		Type:        "docker",
		DockerID:    c.ID,
		DockerName:  name,
		Enabled:     true,
		Public:      labels[d.labelPrefix+".public"] == "true",
	}

	if svc.Category == "" {
		svc.Category = "uncategorized"
	}
	if svc.HealthPath == "" {
		svc.HealthPath = "/health"
	}

	// Determine the actual Docker container name for DNS resolution
	containerName := ""
	for _, n := range c.Names {
		containerName = strings.TrimPrefix(n, "/")
		break
	}

	// Determine the port: prefer 4rch3.io.port label, fall back to first published port
	var privatePort uint16
	if p := labels[d.labelPrefix+".port"]; p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			privatePort = uint16(val)
		}
	}
	if privatePort == 0 && len(c.Ports) > 0 {
		privatePort = c.Ports[0].PrivatePort
	}

	if containerName != "" && privatePort > 0 {
		svc.URL = fmt.Sprintf("http://%s:%d", containerName, privatePort)
	}

	if c.NetworkSettings != nil {
		if netCfg, ok := c.NetworkSettings.Networks[d.network]; ok && netCfg.IPAddress != "" {
			svc.InternalURL = fmt.Sprintf("http://%s", netCfg.IPAddress)
			if privatePort > 0 {
				svc.InternalURL = fmt.Sprintf("http://%s:%d", netCfg.IPAddress, privatePort)
			}
		}
	}

	return svc
}

func (d *DockerDiscovery) Close() error {
	return nil
}

func (d *DockerDiscovery) TestConnection() error {
	req, err := http.NewRequest("GET", "http://localhost/_ping", nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("docker socket not reachable at %s: %w", d.socketPath, err)
	}
	resp.Body.Close()
	return nil
}
