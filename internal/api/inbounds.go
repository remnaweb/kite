package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"panelvpn/internal/db"
	"panelvpn/internal/fw"
	"panelvpn/internal/xray"
)

type inboundBody struct {
	Remark      string `json:"remark"`
	Enable      *bool  `json:"enable"`
	Listen      string `json:"listen"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Network     string `json:"network"`
	Security    string `json:"security"`
	Dest        string `json:"dest"`
	ServerNames string `json:"serverNames"`
	Fingerprint string `json:"fingerprint"`
	SpiderX     string `json:"spiderX"`
	WSPath      string `json:"wsPath"`
	WSHost      string `json:"wsHost"`
	Sniffing    *bool  `json:"sniffing"`
}

func (s *Server) listInbounds(w http.ResponseWriter, r *http.Request) {
	var items []db.Inbound
	if err := s.DB.Preload("Clients").Order("port").Find(&items).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, items)
}

func (s *Server) getInbound(w http.ResponseWriter, r *http.Request) {
	ib, err := s.loadInbound(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "инбаунд не найден")
		return
	}
	writeOK(w, ib)
}

func (s *Server) createInbound(w http.ResponseWriter, r *http.Request) {
	var body inboundBody
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	ib, err := s.inboundFromBody(body, nil)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.DB.Create(ib).Error; err != nil {
		writeErr(w, http.StatusBadRequest, "не удалось создать инбаунд: "+err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, "инбаунд сохранён, но Xray: "+err.Error())
		return
	}
	fw.OpenTCP(ib.Port)
	writeOK(w, ib)
}

func (s *Server) updateInbound(w http.ResponseWriter, r *http.Request) {
	ib, err := s.loadInbound(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "инбаунд не найден")
		return
	}
	var body inboundBody
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	next, err := s.inboundFromBody(body, ib)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	next.ID = ib.ID
	next.PrivateKey = ib.PrivateKey
	next.PublicKey = ib.PublicKey
	next.ShortIds = ib.ShortIds
	if next.Security == "reality" && next.PrivateKey == "" {
		priv, pub, err := xray.GenerateRealityKeys()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		next.PrivateKey = priv
		next.PublicKey = pub
		next.ShortIds = xray.RandomShortID()
	}
	if err := s.DB.Model(ib).Select("*").Updates(next).Error; err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, "сохранено, но Xray: "+err.Error())
		return
	}
	updated, _ := s.loadInbound(strconv.Itoa(int(ib.ID)))
	writeOK(w, updated)
}

func (s *Server) deleteInbound(w http.ResponseWriter, r *http.Request) {
	ib, err := s.loadInbound(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "инбаунд не найден")
		return
	}
	if err := s.DB.Select("Clients").Delete(ib).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, "удалён, но Xray: "+err.Error())
		return
	}
	writeOK(w, nil)
}

func (s *Server) inboundFromBody(body inboundBody, existing *db.Inbound) (*db.Inbound, error) {
	if body.Port <= 0 || body.Port > 65535 {
		return nil, errMsg("порт должен быть 1–65535")
	}
	ib := db.Inbound{
		Remark:      strings.TrimSpace(body.Remark),
		Enable:      true,
		Listen:      xray.FirstNonEmpty(body.Listen, "0.0.0.0"),
		Port:        body.Port,
		Protocol:    xray.FirstNonEmpty(body.Protocol, "vless"),
		Network:     xray.FirstNonEmpty(body.Network, "tcp"),
		Security:    xray.FirstNonEmpty(body.Security, "reality"),
		Dest:        body.Dest,
		ServerNames: body.ServerNames,
		Fingerprint: xray.FirstNonEmpty(body.Fingerprint, "chrome"),
		SpiderX:     xray.FirstNonEmpty(body.SpiderX, "/"),
		WSPath:      xray.FirstNonEmpty(body.WSPath, "/"),
		WSHost:      body.WSHost,
		Sniffing:    true,
	}
	if body.Enable != nil {
		ib.Enable = *body.Enable
	}
	if body.Sniffing != nil {
		ib.Sniffing = *body.Sniffing
	}
	if ib.Remark == "" {
		ib.Remark = "inbound-" + strconv.Itoa(ib.Port)
	}
	if ib.Network == "tcp" && ib.Security == "reality" {
		ib.Dest = xray.FirstNonEmpty(ib.Dest, "www.cloudflare.com:443")
		ib.ServerNames = xray.FirstNonEmpty(ib.ServerNames, "www.cloudflare.com")
		if existing == nil {
			priv, pub, err := xray.GenerateRealityKeys()
			if err != nil {
				return nil, err
			}
			ib.PrivateKey = priv
			ib.PublicKey = pub
			ib.ShortIds = xray.RandomShortID()
		}
	}
	if ib.Network == "ws" && ib.Security == "" {
		ib.Security = "none"
	}
	return &ib, nil
}

func (s *Server) ApplyOnStart() error {
	return s.applyXray()
}

func (s *Server) loadInbound(id string) (*db.Inbound, error) {
	var ib db.Inbound
	if err := s.DB.Preload("Clients").First(&ib, id).Error; err != nil {
		return nil, err
	}
	return &ib, nil
}

func (s *Server) applyXray() error {
	return xray.Apply(s.DB, s.Xray)
}

type clientBody struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	UUID       string   `json:"uuid"`
	Enable     *bool    `json:"enable"`
	ExpiryTime int64    `json:"expiryTime"`
	TotalGB    *float64 `json:"totalGB"`
	TotalBytes *int64   `json:"totalBytes"`
}

func (s *Server) createClient(w http.ResponseWriter, r *http.Request) {
	ib, err := s.loadInbound(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "инбаунд не найден")
		return
	}
	var body clientBody
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	c := s.clientFromBody(body)
	c.InboundID = ib.ID
	if err := s.DB.Create(c).Error; err != nil {
		writeErr(w, http.StatusBadRequest, "не удалось создать клиента: "+err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, "клиент сохранён, но Xray: "+err.Error())
		return
	}
	writeOK(w, c)
}

func (s *Server) updateClient(w http.ResponseWriter, r *http.Request) {
	var c db.Client
	if err := s.DB.First(&c, chi.URLParam(r, "id")).Error; err != nil {
		writeErr(w, http.StatusNotFound, "клиент не найден")
		return
	}
	var body clientBody
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "некорректный запрос")
		return
	}
	next := s.clientFromBody(body)
	next.ID = c.ID
	next.InboundID = c.InboundID
	next.Up = c.Up
	next.Down = c.Down
	next.SubID = c.SubID
	if next.UUID == "" {
		next.UUID = c.UUID
	}
	if next.Email == "" {
		next.Email = c.Email
	}
	if err := s.DB.Model(&c).Select("*").Updates(next).Error; err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, "сохранено, но Xray: "+err.Error())
		return
	}
	_ = s.DB.First(&c, c.ID)
	writeOK(w, c)
}

func (s *Server) deleteClient(w http.ResponseWriter, r *http.Request) {
	var c db.Client
	if err := s.DB.First(&c, chi.URLParam(r, "id")).Error; err != nil {
		writeErr(w, http.StatusNotFound, "клиент не найден")
		return
	}
	if err := s.DB.Delete(&c).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, "удалён, но Xray: "+err.Error())
		return
	}
	writeOK(w, nil)
}

func (s *Server) resetTraffic(w http.ResponseWriter, r *http.Request) {
	var c db.Client
	if err := s.DB.First(&c, chi.URLParam(r, "id")).Error; err != nil {
		writeErr(w, http.StatusNotFound, "клиент не найден")
		return
	}
	if err := s.DB.Model(&c).Updates(map[string]any{"up": 0, "down": 0}).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.applyXray(); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = s.DB.First(&c, c.ID)
	writeOK(w, c)
}

func (s *Server) clientShare(w http.ResponseWriter, r *http.Request) {
	var c db.Client
	if err := s.DB.Preload("Inbound").First(&c, chi.URLParam(r, "id")).Error; err != nil || c.Inbound == nil {
		writeErr(w, http.StatusNotFound, "клиент не найден")
		return
	}
	host, port := s.publicAddr(r, c.Inbound.Port)
	link := xray.ShareLink(*c.Inbound, c, host, port)
	writeOK(w, map[string]any{
		"link": link,
		"sub":  "/sub/" + c.SubID,
		"host": host,
		"port": port,
		"uuid": c.UUID,
	})
}

func (s *Server) clientFromBody(body clientBody) *db.Client {
	id := strings.TrimSpace(body.UUID)
	if id == "" {
		id = uuid.NewString()
	}
	email := strings.TrimSpace(body.Email)
	if email == "" {
		email = strings.ToLower(strings.ReplaceAll(id, "-", "")[:10]) + "@panel"
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = email
	}
	c := &db.Client{
		UUID:       id,
		Email:      email,
		Name:       name,
		Enable:     true,
		ExpiryTime: body.ExpiryTime,
		SubID:      xray.RandomHex(8),
	}
	if body.Enable != nil {
		c.Enable = *body.Enable
	}
	if body.TotalBytes != nil {
		c.TotalBytes = *body.TotalBytes
	} else if body.TotalGB != nil {
		c.TotalBytes = int64(*body.TotalGB * 1024 * 1024 * 1024)
	}
	return c
}

func (s *Server) publicAddr(r *http.Request, inboundPort int) (string, int) {
	host := db.GetSetting(s.DB, "public_host", "")
	if host == "" {
		host = r.Host
		if h, _, ok := strings.Cut(host, ":"); ok {
			host = h
		}
	}
	port := xray.ParsePublicPort(db.GetSetting(s.DB, "public_port", ""), inboundPort)
	return host, port
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func errMsg(s string) error { return simpleError(s) }
