package monitor

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/4rch3/portal/internal/database"
	"github.com/4rch3/portal/internal/metrics"
	"github.com/4rch3/portal/internal/models"
)

type Monitor struct {
	db       *database.DB
	client   *http.Client
	interval time.Duration
	timeout  time.Duration
	retries  int
	mu       sync.Mutex
}

func New(db *database.DB, interval, timeout time.Duration, retries int) *Monitor {
	return &Monitor{
		db:       db,
		interval: interval,
		timeout:  timeout,
		retries:  retries,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        20,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
			},
		},
	}
}

func (m *Monitor) Start(ctx context.Context) {
	slog.Info("starting health monitor", "interval", m.interval, "timeout", m.timeout)

	m.CheckAll(ctx)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.CheckAll(ctx)
		case <-ctx.Done():
			slog.Info("health monitor stopped")
			return
		}
	}
}

func (m *Monitor) CheckAll(ctx context.Context) {
	services, err := m.db.ListServices()
	if err != nil {
		slog.Error("health monitor: list services error", "error", err)
		return
	}

	var wg sync.WaitGroup
	for _, svc := range services {
		if !svc.Enabled {
			continue
		}
		wg.Add(1)
		go func(s *models.Service) {
			defer wg.Done()
			m.CheckService(ctx, s)
		}(svc)
	}
	wg.Wait()
}

func (m *Monitor) CheckService(ctx context.Context, svc *models.Service) {
	target := svc.URL
	if svc.InternalURL != "" {
		target = svc.InternalURL
	}
	if svc.HealthPath != "" {
		target += svc.HealthPath
	}

	var lastStatus string
	var lastLatency int64
	var lastCode int
	var lastErr string

	for attempt := 0; attempt <= m.retries; attempt++ {
		start := time.Now()
		resp, err := m.client.Get(target)
		latency := time.Since(start).Milliseconds()

		if err != nil {
			lastStatus = "offline"
			lastLatency = latency
			lastErr = err.Error()
			slog.Warn("health monitor: service offline",
					"service", svc.Name,
					"error", err,
					"attempt", attempt+1,
					"max_attempts", m.retries+1,
				)
			if attempt < m.retries {
				time.Sleep(1 * time.Second)
				continue
			}
			break
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		lastLatency = latency
		lastCode = resp.StatusCode

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 400:
			lastStatus = "online"
		case resp.StatusCode >= 400 && resp.StatusCode < 500:
			lastStatus = "degraded"
			lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
		default:
			lastStatus = "offline"
			lastErr = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}

		if lastStatus != "offline" {
			break
		}
	}

	metrics.HealthCheckResults.WithLabelValues(svc.Name, lastStatus).Inc()

	if err := m.db.UpdateServiceStatus(svc.ID, lastStatus, lastLatency); err != nil {
		slog.Error("health monitor: status update error", "service", svc.Name, "error", err)
	}

	m.db.RecordHealthCheck(&models.HealthCheck{
		ServiceID: svc.ID,
		Status:    lastStatus,
		Latency:   lastLatency,
		Code:      lastCode,
		Error:     lastErr,
	})
}
