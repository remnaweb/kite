package xray

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/curve25519"
	"panelvpn/internal/db"
)

func GenerateRealityKeys() (privateKey, publicKey string, err error) {
	var priv [32]byte
	if _, err = rand.Read(priv[:]); err != nil {
		return "", "", err
	}
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	pub, err := curve25519.X25519(priv[:], curve25519.Basepoint)
	if err != nil {
		return "", "", err
	}
	return base64.RawURLEncoding.EncodeToString(priv[:]),
		base64.RawURLEncoding.EncodeToString(pub),
		nil
}

func RandomShortID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func RandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func SplitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func BuildConfig(inbounds []db.Inbound, apiPort int) map[string]any {
	xInbounds := []any{
		map[string]any{
			"tag":      "api",
			"listen":   "127.0.0.1",
			"port":     apiPort,
			"protocol": "dokodemo-door",
			"settings": map[string]any{"address": "127.0.0.1"},
		},
	}
	for i := range inbounds {
		ib := inbounds[i]
		if !ib.Enable {
			continue
		}
		xInbounds = append(xInbounds, inboundToXray(ib, i))
	}
	return map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"api": map[string]any{
			"tag":      "api",
			"services": []string{"HandlerService", "LoggerService", "StatsService"},
		},
		"stats": map[string]any{},
		"policy": map[string]any{
			"levels": map[string]any{
				"0": map[string]any{
					"statsUserUplink":   true,
					"statsUserDownlink": true,
				},
			},
			"system": map[string]any{
				"statsInboundUplink":   true,
				"statsInboundDownlink": true,
			},
		},
		"inbounds": xInbounds,
		"outbounds": []any{
			map[string]any{"protocol": "freedom", "tag": "direct"},
			map[string]any{"protocol": "blackhole", "tag": "blocked"},
		},
		"routing": map[string]any{
			"domainStrategy": "AsIs",
			"rules": []any{
				map[string]any{
					"type":        "field",
					"inboundTag":  []string{"api"},
					"outboundTag": "api",
				},
			},
		},
	}
}

func inboundToXray(ib db.Inbound, idx int) map[string]any {
	clients := make([]any, 0)
	for _, c := range ib.Clients {
		if !c.ActiveForXray() {
			continue
		}
		entry := map[string]any{
			"id":    c.UUID,
			"email": c.Email,
		}
		if ib.Security == "reality" && ib.Network == "tcp" {
			entry["flow"] = "xtls-rprx-vision"
		}
		clients = append(clients, entry)
	}

	settings := map[string]any{
		"clients":    clients,
		"decryption": "none",
	}

	stream := map[string]any{
		"network":  ib.Network,
		"security": ib.Security,
	}
	if ib.Security == "reality" {
		shortIDs := SplitCSV(ib.ShortIds)
		if len(shortIDs) == 0 {
			shortIDs = []string{""}
		}
		stream["realitySettings"] = map[string]any{
			"show":        false,
			"dest":        FirstNonEmpty(ib.Dest, "www.cloudflare.com:443"),
			"xver":        0,
			"serverNames": defaultServerNames(ib.ServerNames),
			"privateKey":  ib.PrivateKey,
			"shortIds":    shortIDs,
		}
	}
	if ib.Network == "ws" {
		ws := map[string]any{
			"path": FirstNonEmpty(ib.WSPath, "/"),
		}
		if ib.WSHost != "" {
			ws["host"] = ib.WSHost
		}
		stream["wsSettings"] = ws
		if ib.Security == "" {
			stream["security"] = "none"
		}
	}
	if ib.Security == "none" || ib.Security == "" {
		stream["security"] = "none"
	}

	tag := fmt.Sprintf("in-%d-%d", ib.ID, ib.Port)
	if idx >= 0 && ib.ID == 0 {
		tag = fmt.Sprintf("in-new-%d", ib.Port)
	}

	obj := map[string]any{
		"tag":            tag,
		"listen":         FirstNonEmpty(ib.Listen, "0.0.0.0"),
		"port":           ib.Port,
		"protocol":       FirstNonEmpty(ib.Protocol, "vless"),
		"settings":       settings,
		"streamSettings": stream,
	}
	if ib.Sniffing {
		obj["sniffing"] = map[string]any{
			"enabled":      true,
			"destOverride": []string{"http", "tls", "quic"},
			"routeOnly":    true,
		}
	}
	return obj
}

func defaultServerNames(raw string) []string {
	names := SplitCSV(raw)
	if len(names) == 0 {
		return []string{"www.cloudflare.com"}
	}
	return names
}
