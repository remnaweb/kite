package api

import (
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"panelvpn/internal/db"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := jsonDecode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	var admin db.Admin
	if err := s.DB.Where("username = ?", body.Username).First(&admin).Error; err != nil {
		writeErr(w, http.StatusUnauthorized, "неверный логин или пароль")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(body.Password)); err != nil {
		writeErr(w, http.StatusUnauthorized, "неверный логин или пароль")
		return
	}
	token := newToken()
	s.mu.Lock()
	s.sessions[token] = session{AdminID: admin.ID, Expiry: time.Now().Add(7 * 24 * time.Hour)}
	s.mu.Unlock()
	setSessionCookie(w, token, 7*24*3600)
	writeOK(w, map[string]any{"username": admin.Username})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("panel_session"); err == nil {
		s.mu.Lock()
		delete(s.sessions, c.Value)
		s.mu.Unlock()
	}
	setSessionCookie(w, "", -1)
	writeOK(w, nil)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	admin := s.adminFrom(r)
	if admin == nil {
		writeErr(w, http.StatusUnauthorized, "нет сессии")
		return
	}
	writeOK(w, map[string]any{"id": admin.ID, "username": admin.Username})
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	admin := s.adminFrom(r)
	if admin == nil {
		writeErr(w, http.StatusUnauthorized, "нет сессии")
		return
	}
	var body struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := jsonDecode(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	if len(body.Next) < 4 {
		writeErr(w, http.StatusBadRequest, "пароль слишком короткий")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(body.Current)); err != nil {
		writeErr(w, http.StatusBadRequest, "текущий пароль неверный")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.Next), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "не удалось сохранить пароль")
		return
	}
	if err := s.DB.Model(admin).Update("password_hash", string(hash)).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, nil)
}

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("panel_session")
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "нужна авторизация")
			return
		}
		s.mu.Lock()
		sess, ok := s.sessions[c.Value]
		if ok && time.Now().After(sess.Expiry) {
			delete(s.sessions, c.Value)
			ok = false
		}
		s.mu.Unlock()
		if !ok {
			writeErr(w, http.StatusUnauthorized, "нужна авторизация")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) adminFrom(r *http.Request) *db.Admin {
	c, err := r.Cookie("panel_session")
	if err != nil {
		return nil
	}
	s.mu.Lock()
	sess, ok := s.sessions[c.Value]
	s.mu.Unlock()
	if !ok {
		return nil
	}
	var admin db.Admin
	if err := s.DB.First(&admin, sess.AdminID).Error; err != nil {
		return nil
	}
	return &admin
}

func jsonDecode(r *http.Request, dst any) error {
	return readJSON(r, dst)
}
