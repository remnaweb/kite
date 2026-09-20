package api

import (
	"log"
	"time"

	"panelvpn/internal/db"
	"panelvpn/internal/xray"
)

func (s *Server) CollectLoop(stop <-chan struct{}) {
	t := time.NewTicker(12 * time.Second)
	defer t.Stop()
	s.collectOnce()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			s.collectOnce()
		}
	}
}

func (s *Server) collectOnce() {
	if !s.Xray.Running() {
		return
	}
	stats, err := s.Xray.QueryStats(true)
	if err != nil {
		log.Printf("xray stats: %v", err)
		return
	}
	deltas := xray.UserTrafficDeltas(stats)
	if len(deltas) == 0 {
		return
	}
	changed := false
	for email, td := range deltas {
		if td[0] == 0 && td[1] == 0 {
			continue
		}
		var c db.Client
		if err := s.DB.Where("email = ?", email).First(&c).Error; err != nil {
			continue
		}
		if err := s.DB.Model(&c).Updates(map[string]any{
			"up":   c.Up + td[0],
			"down": c.Down + td[1],
		}).Error; err != nil {
			log.Printf("traffic update %s: %v", email, err)
			continue
		}
		c.Up += td[0]
		c.Down += td[1]
		if c.Exhausted() && c.Enable {
			changed = true
		}
	}
	if changed {
		if err := s.applyXray(); err != nil {
			log.Printf("reapply after traffic limit: %v", err)
		}
	}
}
