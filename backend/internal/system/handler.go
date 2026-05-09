// Package system — métricas do host: CPU, RAM, temperatura, disco, rede, uptime.
//
// Usa gopsutil para portabilidade e robustez (já lida com diferenças entre
// kernels). No Raspberry Pi, a temperatura vem de /sys/class/thermal/thermal_zone0/temp.
package system

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"

	"github.com/pinas/pinas/internal/storage"
)

type Stats struct {
	Hostname    string    `json:"hostname"`
	OS          string    `json:"os"`
	Kernel      string    `json:"kernel"`
	Arch        string    `json:"arch"`
	Uptime      uint64    `json:"uptime_sec"`
	CPUPercent  float64   `json:"cpu_percent"`
	CPUCores    int       `json:"cpu_cores"`
	LoadAvg     [3]float64 `json:"load_avg"`
	TempCelsius float64   `json:"temp_celsius"` // 0 se não disponível
	Memory      MemStats  `json:"memory"`
	Disk        storage.Usage `json:"disk"`
	Network     []NetStat `json:"network"`
	Timestamp   int64     `json:"timestamp"`
}

type MemStats struct {
	TotalBytes     uint64  `json:"total"`
	UsedBytes      uint64  `json:"used"`
	FreeBytes      uint64  `json:"free"`
	AvailableBytes uint64  `json:"available"`
	UsedPercent    float64 `json:"used_percent"`
}

type NetStat struct {
	Name     string `json:"name"`
	BytesIn  uint64 `json:"bytes_in"`
	BytesOut uint64 `json:"bytes_out"`
}

type Handler struct {
	jail *storage.Jail
}

func NewHandler(jail *storage.Jail) *Handler {
	return &Handler{jail: jail}
}

func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	s := Stats{
		Arch:      runtime.GOARCH,
		Timestamp: time.Now().Unix(),
		CPUCores:  runtime.NumCPU(),
	}

	if hostInfo, err := host.InfoWithContext(ctx); err == nil {
		s.Hostname = hostInfo.Hostname
		s.OS = hostInfo.Platform + " " + hostInfo.PlatformVersion
		s.Kernel = hostInfo.KernelVersion
		s.Uptime = hostInfo.Uptime
	}

	if pcs, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false); err == nil && len(pcs) > 0 {
		s.CPUPercent = pcs[0]
	}

	if avg, err := load.AvgWithContext(ctx); err == nil {
		s.LoadAvg = [3]float64{avg.Load1, avg.Load5, avg.Load15}
	}

	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil {
		s.Memory = MemStats{
			TotalBytes:     vm.Total,
			UsedBytes:      vm.Used,
			FreeBytes:      vm.Free,
			AvailableBytes: vm.Available,
			UsedPercent:    vm.UsedPercent,
		}
	}

	if u, err := h.jail.DiskUsage(); err == nil {
		s.Disk = u
	}

	s.TempCelsius = readPiTemperature()

	if nets, err := net.IOCountersWithContext(ctx, true); err == nil {
		for _, n := range nets {
			if n.Name == "lo" || strings.HasPrefix(n.Name, "veth") || strings.HasPrefix(n.Name, "docker") || strings.HasPrefix(n.Name, "br-") {
				continue
			}
			s.Network = append(s.Network, NetStat{
				Name:     n.Name,
				BytesIn:  n.BytesRecv,
				BytesOut: n.BytesSent,
			})
		}
	}

	writeJSON(w, http.StatusOK, s)
}

// readPiTemperature lê /sys/class/thermal/thermal_zone0/temp.
// O arquivo retorna a temp em milicentigrados (ex: "48562" = 48.562°C).
// No container, este path precisa estar montado (em geral está, /sys já é exposto).
func readPiTemperature() float64 {
	candidates := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/host/sys/class/thermal/thermal_zone0/temp",
	}
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		raw := strings.TrimSpace(string(data))
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		return v / 1000.0
	}
	return 0
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
