package xray

import (
	"gorm.io/gorm"
	"panelvpn/internal/db"
)

func Apply(conn *gorm.DB, m *Manager) error {
	var inbounds []db.Inbound
	if err := conn.Preload("Clients").Find(&inbounds).Error; err != nil {
		return err
	}
	cfg := BuildConfig(inbounds, m.APIPort)
	if err := m.WriteConfig(cfg); err != nil {
		return err
	}
	if !m.BinaryExists() {
		return nil
	}
	return m.Restart()
}

func WriteOnly(conn *gorm.DB, m *Manager) error {
	var inbounds []db.Inbound
	if err := conn.Preload("Clients").Find(&inbounds).Error; err != nil {
		return err
	}
	return m.WriteConfig(BuildConfig(inbounds, m.APIPort))
}
