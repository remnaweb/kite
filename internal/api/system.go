package api

import (
	"net/http"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"panelvpn/internal/db"
)

func (s *Server) systemStatus(w http.ResponseWriter, r *http.Request) {
	cpuPct := 0.0
	if v, err := cpu.Percent(120*time.Millisecond, false); err == nil && len(v) > 0 {
		cpuPct = v[0]
	}
	vm, _ := mem.VirtualMemory()
	du, _ := disk.Usage("/")
	hi, _ := host.Info()
	var up, down uint64
	if counters, err := net.IOCounters(false); err == nil && len(counters) > 0 {
		up = counters[0].BytesSent
		down = counters[0].BytesRecv
	}

	var inbounds, clients, active int64
	_ = s.DB.Model(&db.Inbound{}).Count(&inbounds).Error
	_ = s.DB.Model(&db.Client{}).Count(&clients).Error
	now := time.Now().UnixMilli()
	_ = s.DB.Model(&db.Client{}).Where("enable = ? AND (expiry_time = 0 OR expiry_time > ?) AND (total_bytes = 0 OR up + down < total_bytes)", true, now).Count(&active).Error

	var traffic struct{ Up, Down int64 }
	_ = s.DB.Model(&db.Client{}).Select("COALESCE(SUM(up),0) as up, COALESCE(SUM(down),0) as down").Scan(&traffic)

	memUsed, memTotal := uint64(0), uint64(0)
	if vm != nil {
		memUsed, memTotal = vm.Used, vm.Total
	}
	diskUsed, diskTotal := uint64(0), uint64(0)
	if du != nil {
		diskUsed, diskTotal = du.Used, du.Total
	}
	hostName, osName, uptime := "", runtime.GOOS, uint64(0)
	if hi != nil {
		hostName = hi.Hostname
		osName = hi.Platform + " " + hi.PlatformVersion
		uptime = hi.Uptime
	}

	writeOK(w, map[string]any{
		"cpu":          cpuPct,
		"memUsed":      memUsed,
		"memTotal":     memTotal,
		"diskUsed":     diskUsed,
		"diskTotal":    diskTotal,
		"netUp":        up,
		"netDown":      down,
		"hostname":     hostName,
		"os":           osName,
		"uptime":       uptime,
		"panelUptime":  time.Since(s.Started).Seconds(),
		"goos":         runtime.GOOS,
		"goarch":       runtime.GOARCH,
		"inbounds":     inbounds,
		"clients":      clients,
		"active":       active,
		"trafficUp":    traffic.Up,
		"trafficDown":  traffic.Down,
		"xray":         s.Xray.Status(),
		"xrayVersion":  s.Xray.Version(),
	})
}

func (s *Server) xrayStatus(w http.ResponseWriter, r *http.Request) {
	st := s.Xray.Status()
	st["version"] = s.Xray.Version()
	writeOK(w, st)
}

func (s *Server) xrayRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, s.Xray.Status())
}

func (s *Server) xrayStop(w http.ResponseWriter, r *http.Request) {
	if err := s.Xray.Stop(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, s.Xray.Status())
}

func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"publicHost": db.GetSetting(s.DB, "public_host", ""),
		"publicPort": db.GetSetting(s.DB, "public_port", ""),
		"panelTitle": db.GetSetting(s.DB, "panel_title", "Kite"),
	})
}

func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PublicHost string `json:"publicHost"`
		PublicPort string `json:"publicPort"`
		PanelTitle string `json:"panelTitle"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	_ = db.SetSetting(s.DB, "public_host", body.PublicHost)
	_ = db.SetSetting(s.DB, "public_port", body.PublicPort)
	if body.PanelTitle != "" {
		_ = db.SetSetting(s.DB, "panel_title", body.PanelTitle)
	}
	s.getSettings(w, r)
}
