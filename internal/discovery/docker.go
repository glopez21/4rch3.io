package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
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
	log.Printf("[discovery] starting docker discovery (socket: %s, interval: %s)", d.socketPath, d.interval)

	d.Scan(ctx)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.Scan(ctx)
		case <-ctx.Done():
			log.Printf("[discovery] stopping")
			return
		}
	}
}

type dockerContainer struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	Image  string            `json:"Image"`
	State  string            `json:"State"`
	Status string            `json:"Status"`
	Labels map[string]string `json:"Labels"`
	Ports  []dockerPort      `json:"Ports"`
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
	filter := fmt.Sprintf(`{"label":{"%s.enable":["true"]}}`, d.labelPrefix)
	url := fmt.Sprintf("http://localhost/containers/json?filters=%s", filter)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.Printf("[discovery] error creating request: %v", err)
		return
	}

	resp, err := d.client.Do(req)
	if err != nil {
		log.Printf("[discovery] error connecting to docker socket: %v", err)
		return
	}
	defer resp.Body.Close()

	var containers []dockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		log.Printf("[discovery] error decoding response: %v", err)
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
			log.Printf("[discovery] error saving service %s: %v", svc.Name, err)
			continue
		}
		log.Printf("[discovery] discovered service: %s (id=%d)", svc.Name, id)
	}

	if len(containers) == 0 {
		log.Printf("[discovery] no running containers found with label %s.enable=true", d.labelPrefix)
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

	if len(c.Ports) > 0 {
		port := c.Ports[0]
		svc.URL = fmt.Sprintf("http://%s:%d", name, port.PrivatePort)
	}

	if c.NetworkSettings != nil {
		if netCfg, ok := c.NetworkSettings.Networks[d.network]; ok && netCfg.IPAddress != "" {
			svc.InternalURL = fmt.Sprintf("http://%s", netCfg.IPAddress)
			if len(c.Ports) > 0 {
				svc.InternalURL = fmt.Sprintf("http://%s:%d", netCfg.IPAddress, c.Ports[0].PrivatePort)
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
