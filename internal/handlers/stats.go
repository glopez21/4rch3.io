package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/4rch3/portal/internal/database"
)

type dockerStats struct {
	Read     time.Time             `json:"read"`
	PreCPU   cpuSection            `json:"precpu_stats"`
	CPU      cpuSection            `json:"cpu_stats"`
	Memory   memSection            `json:"memory_stats"`
	Networks map[string]netSection `json:"networks"`
}
type cpuSection struct {
	CPUUsage struct {
		TotalUsage uint64 `json:"total_usage"`
	} `json:"cpu_usage"`
	SystemCPUUsage uint64 `json:"system_cpu_usage"`
	OnlineCPUs     int    `json:"online_cpus"`
}
type memSection struct {
	Usage uint64 `json:"usage"`
	Limit uint64 `json:"limit"`
}
type netSection struct {
	RxBytes uint64 `json:"rx_bytes"`
	TxBytes uint64 `json:"tx_bytes"`
}

// ServiceStats streams SSE of cpu/mem/net for a service
func (h *DashboardHandler) ServiceStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("id")
	if q == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	var id int64
	for i := 0; i < len(q); i++ {
		if q[i] < '0' || q[i] > '9' {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
	}
	for i := 0; i < len(q); i++ {
		id = id*10 + int64(q[i]-'0')
	}

	svc, err := h.db.GetService(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if svc.DockerID == "" {
		http.Error(w, "no docker source", http.StatusBadRequest)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}

	// unix socket http client
	socket := h.dockerSocket
	if socket == "" {
		socket = "/var/run/docker.sock"
	}
	client := &http.Client{Timeout: 0, Transport: &http.Transport{DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
		return net.DialTimeout("unix", socket, 5*time.Second)
	}}}

	req, _ := http.NewRequestWithContext(r.Context(), "GET", fmt.Sprintf("http://localhost/containers/%s/stats?stream=1", svc.DockerID), nil)
	resp, err := client.Do(req)
	if err != nil {
		slog.Warn("stats: socket error", "error", err)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)

	// previous counters for deltas
	var prev dockerStats
	var havePrev bool

	enc := func(ts int64, cpu float64, memUsed, memLimit uint64, rx, tx float64) {
		// send one SSE event
		payload := struct {
			TS       int64   `json:"ts"`
			CPU      float64 `json:"cpu"`
			MemUsed  uint64  `json:"mem_used"`
			MemLimit uint64  `json:"mem_limit"`
			RX       float64 `json:"rx"`
			TX       float64 `json:"tx"`
		}{TS: ts, CPU: cpu, MemUsed: memUsed, MemLimit: memLimit, RX: rx, TX: tx}
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "data: %s\n\n", b)
		flusher.Flush()
	}

	dec := json.NewDecoder(reader)
	ticker := time.NewTicker(1100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		default:
		}
		var st dockerStats
		if err := dec.Decode(&st); err != nil {
			return
		}
		if !havePrev {
			prev = st
			havePrev = true
			continue
		}
		// compute cpu%
		cpuDelta := float64(st.CPU.CPUUsage.TotalUsage - prev.CPU.CPUUsage.TotalUsage)
		sysDelta := float64(st.CPU.SystemCPUUsage - prev.CPU.SystemCPUUsage)
		online := st.CPU.OnlineCPUs
		if online == 0 {
			online = 1
		}
		cpuPct := 0.0
		if cpuDelta > 0 && sysDelta > 0 {
			cpuPct = (cpuDelta / sysDelta) * float64(online) * 100.0
		}

		// mem
		memUsed := st.Memory.Usage
		memLimit := st.Memory.Limit

		// net rates bytes/sec
		var rxCur, txCur uint64
		for _, n := range st.Networks {
			rxCur += n.RxBytes
			txCur += n.TxBytes
		}
		var rxPrev, txPrev uint64
		for _, n := range prev.Networks {
			rxPrev += n.RxBytes
			txPrev += n.TxBytes
		}
		dt := st.Read.Sub(prev.Read).Seconds()
		if dt <= 0 {
			dt = 1
		}
		rxRate := float64(rxCur-rxPrev) / dt
		txRate := float64(txCur-txPrev) / dt

		enc(st.Read.Unix(), cpuPct, memUsed, memLimit, rxRate, txRate)
		prev = st
		<-ticker.C
	}
}

// ensure compile-time deps
var _ = database.DB{}
