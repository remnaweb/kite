package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"

	"panelvpn/internal/webui"
	"panelvpn/internal/xray"
)

type Server struct {
	DB      *gorm.DB
	Xray    *xray.Manager
	Started time.Time

	mu       sync.Mutex
	sessions map[string]session
}

type session struct {
	AdminID uint
	Expiry  time.Time
}

func New(db *gorm.DB, xm *xray.Manager) *Server {
	return &Server{
		DB:       db,
		Xray:     xm,
		Started:  time.Now(),
		sessions: map[string]session{},
	}
}

func (s *Server) Router(webDist string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Post("/api/login", s.login)

	r.Group(func(r chi.Router) {
		r.Use(s.requireAuth)
		r.Post("/api/logout", s.logout)
		r.Get("/api/me", s.me)
		r.Post("/api/me/password", s.changePassword)

		r.Get("/api/system", s.systemStatus)
		r.Get("/api/settings", s.getSettings)
		r.Put("/api/settings", s.updateSettings)

		r.Get("/api/xray", s.xrayStatus)
		r.Post("/api/xray/restart", s.xrayRestart)
		r.Post("/api/xray/stop", s.xrayStop)

		r.Get("/api/inbounds", s.listInbounds)
		r.Post("/api/inbounds", s.createInbound)
		r.Get("/api/inbounds/{id}", s.getInbound)
		r.Put("/api/inbounds/{id}", s.updateInbound)
		r.Delete("/api/inbounds/{id}", s.deleteInbound)

		r.Post("/api/inbounds/{id}/clients", s.createClient)
		r.Put("/api/clients/{id}", s.updateClient)
		r.Delete("/api/clients/{id}", s.deleteClient)
		r.Post("/api/clients/{id}/reset-traffic", s.resetTraffic)
		r.Get("/api/clients/{id}/share", s.clientShare)
	})

	r.Get("/sub/{id}", s.subscription)

	if webDist != "" {
		if st, err := os.Stat(webDist); err == nil && st.IsDir() {
			r.Handle("/*", spaHandler(webDist))
			return r
		}
	}
	if sub, err := fs.Sub(webui.Dist, "dist"); err == nil {
		r.Handle("/*", spaFSHandler(sub))
	}
	return r
}

func spaHandler(dir string) http.Handler {
	root := http.Dir(dir)
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := root.Open(path)
		if err != nil {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		stat, err := f.Stat()
		_ = f.Close()
		if err != nil || stat.IsDir() {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func spaFSHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := root.Open(path)
		if err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, r)
			return
		}
		stat, err := f.Stat()
		_ = f.Close()
		if err != nil || stat.IsDir() {
			r = r.Clone(r.Context())
			r.URL.Path = "/index.html"
			fileServer.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{"success": false, "message": msg})
}

func writeOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "data": data})
}

func readJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	return dec.Decode(dst)
}

func newToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func setSessionCookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "panel_session",
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
