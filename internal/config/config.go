package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Listen      string
	DataDir     string
	XrayBin     string
	XrayAPIPort int
	WebDist     string
	EnvFile     string
}

func Load() Config {
	envFile := getenv("PANEL_ENV", "/etc/panelvpn/panel.env")
	loadEnvFile(envFile)
	dataDir := getenv("PANEL_DATA", "data")
	if abs, err := filepath.Abs(dataDir); err == nil {
		dataDir = abs
	}
	return Config{
		Listen:      getenv("PANEL_LISTEN", "0.0.0.0:2053"),
		DataDir:     dataDir,
		XrayBin:     firstExisting(os.Getenv("XRAY_BIN"), "bin/xray", "xray"),
		XrayAPIPort: getenvInt("XRAY_API_PORT", 62789),
		WebDist:     getenv("PANEL_WEB", "web/dist"),
		EnvFile:     envFile,
	}
}

func loadEnvFile(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || os.Getenv(k) != "" {
			continue
		}
		_ = os.Setenv(k, v)
	}
}

func (c Config) DBPath() string {
	return filepath.Join(c.DataDir, "panel.db")
}

func (c Config) XrayConfigPath() string {
	return filepath.Join(c.DataDir, "xray.json")
}

func (c Config) XrayLogPath() string {
	return filepath.Join(c.DataDir, "xray.log")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func firstExisting(paths ...string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	if p, err := lookPath("xray"); err == nil {
		return p
	}
	return "bin/xray"
}

func lookPath(file string) (string, error) {
	return execLookPath(file)
}
