package xray

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"panelvpn/internal/db"
)

func ShareLink(ib db.Inbound, c db.Client, host string, port int) string {
	if port <= 0 {
		port = ib.Port
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}

	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("type", ib.Network)
	q.Set("security", ib.Security)
	if ib.Security == "" {
		q.Set("security", "none")
	}

	if ib.Security == "reality" {
		sni := "www.cloudflare.com"
		if names := SplitCSV(ib.ServerNames); len(names) > 0 {
			sni = names[0]
		}
		sid := ""
		if ids := SplitCSV(ib.ShortIds); len(ids) > 0 {
			sid = ids[0]
		}
		q.Set("pbk", ib.PublicKey)
		q.Set("fp", FirstNonEmpty(ib.Fingerprint, "chrome"))
		q.Set("sni", sni)
		q.Set("sid", sid)
		q.Set("spx", FirstNonEmpty(ib.SpiderX, "/"))
		q.Set("flow", "xtls-rprx-vision")
	}
	if ib.Network == "ws" {
		q.Set("path", FirstNonEmpty(ib.WSPath, "/"))
		if ib.WSHost != "" {
			q.Set("host", ib.WSHost)
		}
	}

	fragment := url.PathEscape(FirstNonEmpty(c.Name, c.Email, ib.Remark))
	return fmt.Sprintf("vless://%s@%s:%d?%s#%s", c.UUID, host, port, q.Encode(), fragment)
}

func ParsePublicPort(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}
