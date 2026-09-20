package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"panelvpn/internal/db"
	"panelvpn/internal/fw"
	"panelvpn/internal/xray"
)

func SetAdmin(conn *gorm.DB, username, password string) error {
	if username == "" || password == "" {
		return fmt.Errorf("логин и пароль обязательны")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	var n int64
	if err := conn.Model(&db.Admin{}).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		return conn.Create(&db.Admin{Username: username, PasswordHash: string(hash)}).Error
	}
	return conn.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Model(&db.Admin{}).
		Updates(map[string]any{"username": username, "password_hash": string(hash)}).Error
}

func FirstAdmin(conn *gorm.DB) (*db.Admin, error) {
	var a db.Admin
	if err := conn.First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func InitReality(conn *gorm.DB, remark string, port int, clientName, publicHost string, openFirewall bool) (*db.Inbound, *db.Client, string, error) {
	if port <= 0 {
		port = 443
	}
	if remark == "" {
		remark = "VLESS-Reality"
	}
	if clientName == "" {
		clientName = "main"
	}
	var exists int64
	_ = conn.Model(&db.Inbound{}).Where("port = ?", port).Count(&exists)
	if exists > 0 {
		return nil, nil, "", fmt.Errorf("порт %d уже занят инбаундом", port)
	}
	priv, pub, err := xray.GenerateRealityKeys()
	if err != nil {
		return nil, nil, "", err
	}
	ib := &db.Inbound{
		Remark:      remark,
		Enable:      true,
		Listen:      "0.0.0.0",
		Port:        port,
		Protocol:    "vless",
		Network:     "tcp",
		Security:    "reality",
		Dest:        "www.cloudflare.com:443",
		ServerNames: "www.cloudflare.com",
		PrivateKey:  priv,
		PublicKey:   pub,
		ShortIds:    xray.RandomShortID(),
		Fingerprint: "chrome",
		SpiderX:     "/",
		Sniffing:    true,
	}
	if err := conn.Create(ib).Error; err != nil {
		return nil, nil, "", err
	}
	id := uuid.NewString()
	c := &db.Client{
		InboundID: ib.ID,
		UUID:      id,
		Email:     clientName + "@panel",
		Name:      clientName,
		Enable:    true,
		SubID:     xray.RandomHex(8),
	}
	if err := conn.Create(c).Error; err != nil {
		return nil, nil, "", err
	}
	if publicHost != "" {
		_ = db.SetSetting(conn, "public_host", publicHost)
	}
	host := db.GetSetting(conn, "public_host", publicHost)
	link := xray.ShareLink(*ib, *c, host, port)
	if openFirewall {
		fw.OpenTCP(port)
	}
	return ib, c, link, nil
}

func WriteEnvFile(path, listen, dataDir, xrayBin string) error {
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}
	if xrayBin != "" {
		if abs, err := filepath.Abs(xrayBin); err == nil {
			xrayBin = abs
		}
	}
	body := fmt.Sprintf("PANEL_LISTEN=%s\nPANEL_DATA=%s\nXRAY_BIN=%s\nPANEL_ENV=%s\n", listen, dataDir, xrayBin, path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o600)
}
