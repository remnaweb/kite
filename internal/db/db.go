package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(path string) (*gorm.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	conn, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	if err := conn.AutoMigrate(&Admin{}, &Setting{}, &Inbound{}, &Client{}); err != nil {
		return nil, err
	}
	if err := seed(conn); err != nil {
		return nil, err
	}
	return conn, nil
}

func seed(db *gorm.DB) error {
	var n int64
	if err := db.Model(&Admin{}).Count(&n).Error; err != nil {
		return err
	}
	if n == 0 {
		user := os.Getenv("PANEL_USERNAME")
		pass := os.Getenv("PANEL_PASSWORD")
		if user == "" {
			user = "admin"
		}
		if pass == "" {
			pass = "admin"
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := db.Create(&Admin{Username: user, PasswordHash: string(hash)}).Error; err != nil {
			return err
		}
	}
	defaults := map[string]string{
		"public_host": "",
		"public_port": "",
		"panel_title": "Kite",
	}
	for k, v := range defaults {
		row := Setting{Key: k, Value: v}
		if err := db.Where("key = ?", k).Attrs(Setting{Value: v}).FirstOrCreate(&row).Error; err != nil {
			return fmt.Errorf("seed setting %s: %w", k, err)
		}
	}
	return nil
}

func GetSetting(db *gorm.DB, key, fallback string) string {
	var s Setting
	if err := db.First(&s, "key = ?", key).Error; err != nil {
		return fallback
	}
	if s.Value == "" {
		return fallback
	}
	return s.Value
}

func SetSetting(db *gorm.DB, key, value string) error {
	s := Setting{Key: key, Value: value}
	return db.Save(&s).Error
}
