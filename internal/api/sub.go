package api

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"panelvpn/internal/db"
	"panelvpn/internal/xray"
)

func (s *Server) subscription(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var c db.Client
	if err := s.DB.Preload("Inbound").Where("sub_id = ?", id).First(&c).Error; err != nil || c.Inbound == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if !c.ActiveForXray() {
		http.Error(w, "disabled", http.StatusForbidden)
		return
	}
	host, port := s.publicAddr(r, c.Inbound.Port)
	link := xray.ShareLink(*c.Inbound, c, host, port)
	body := base64.StdEncoding.EncodeToString([]byte(link + "\n"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Profile-Title", encodeHeader(c.Name))
	_, _ = w.Write([]byte(body))
}

func encodeHeader(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		s = "Kite"
	}
	return "base64:" + base64.StdEncoding.EncodeToString([]byte(s))
}
