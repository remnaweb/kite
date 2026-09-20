#!/usr/bin/env bash
# Kite — как 3X-UI: качает готовый бинарник, на VPS ничего не собирает.
#
#   bash <(curl -fsSL https://github.com/remnaweb/kite/releases/download/nightly/install.sh)
#
set -euo pipefail

red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; plain='\033[0m'
INSTALL_DIR=/usr/local/panelvpn
ENV_DIR=/etc/panelvpn
XRAY_VERSION="${XRAY_VERSION:-v25.8.3}"
KITE_REPO="${KITE_REPO:-remnaweb/kite}"
KITE_RELEASE="${KITE_RELEASE:-nightly}"

log()  { echo -e "${green}[Kite]${plain} $*"; }
warn() { echo -e "${yellow}[Kite]${plain} $*"; }
die()  { echo -e "${red}[Kite]${plain} $*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || die "Нужен root: sudo bash install.sh"
[[ "$(uname -s)" == Linux ]] || die "Скрипт для Linux VPS."
log "installer 4 — без вопросов, логин и пароль сгенерирую сам"

# Старый инсталлятор тащил Go+Node и забивал диск — вычищаем это.
rm -rf /usr/local/kite-src /tmp/go* /tmp/node* /tmp/xray* /tmp/kite* /root/go/pkg
if [[ -d /usr/local/go ]]; then
  warn "убираю Go с прошлой установки (~сотни МБ)…"
  rm -rf /usr/local/go
fi
rm -rf /usr/local/node-v22* /usr/local/bin/node /usr/local/bin/npm 2>/dev/null || true
apt-get clean >/dev/null 2>&1 || true

avail="$(df -Pm / | awk 'NR==2{print $4}')"
if [[ "${avail:-0}" -lt 250 ]]; then
  die "на диске ${avail:-0} MB. Нужно ~300 MB. Посмотри: df -h"
fi

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) GOARCH=amd64; XRAY_ASSET=Xray-linux-64.zip ;;
  aarch64|arm64) GOARCH=arm64; XRAY_ASSET=Xray-linux-arm64-v8a.zip ;;
  *) die "Архитектура $arch не поддерживается" ;;
esac

if [[ -f /etc/os-release ]]; then
  # shellcheck disable=SC1091
  . /etc/os-release
else
  ID=unknown
fi

install_pkgs() {
  log "пакеты…"
  case "${ID:-}" in
    ubuntu|debian|armbian)
      export DEBIAN_FRONTEND=noninteractive
      apt-get update -y
      apt-get install -y --no-install-recommends curl wget tar unzip ca-certificates
      ;;
    centos|rhel|rocky|almalinux|fedora)
      (yum install -y curl wget tar unzip ca-certificates || dnf install -y curl wget tar unzip ca-certificates)
      ;;
    *)
      warn "неизвестный дистрибутив ${ID:-}, ставлю без пакетного менеджера"
      ;;
  esac
}

download_panel() {
  local asset="kite-linux-${GOARCH}.tar.gz"
  local url="https://github.com/${KITE_REPO}/releases/download/${KITE_RELEASE}/${asset}"
  local tmp="/tmp/${asset}"
  log "скачиваю панель (${asset})…"
  local ok=0
  local i
  for i in 1 2 3 4 5 6 7 8 9 10; do
    if curl -fL --retry 2 --retry-delay 2 -o "$tmp" "$url"; then
      ok=1
      break
    fi
    warn "релиз ещё собирается на GitHub, жду 15с (${i}/10)…"
    sleep 15
  done
  [[ "$ok" == "1" ]] || die "не скачался ${url}. Подожди 2–3 минуты после пуша и повтори."
  local unpack
  unpack="$(mktemp -d)"
  tar -xzf "$tmp" -C "$unpack"
  [[ -f "$unpack/panel" ]] || die "в архиве нет panel"
  systemctl stop panelvpn >/dev/null 2>&1 || true
  mkdir -p "$INSTALL_DIR" "$INSTALL_DIR/data" "$INSTALL_DIR/bin" "$ENV_DIR"
  install -m 755 "$unpack/panel" "$INSTALL_DIR/panel"
  if [[ -f "$unpack/panelvpn.sh" ]]; then
    install -m 755 "$unpack/panelvpn.sh" /usr/local/bin/panelvpn
  fi
  if [[ -f "$unpack/panelvpn.service" ]]; then
    cp -f "$unpack/panelvpn.service" /etc/systemd/system/panelvpn.service
  fi
  rm -rf "$tmp" "$unpack"
}

install_xray() {
  log "ставлю Xray ${XRAY_VERSION}"
  mkdir -p "$INSTALL_DIR/bin"
  local tmp=/tmp/xray.zip
  curl -fsSL "https://github.com/XTLS/Xray-core/releases/download/${XRAY_VERSION}/${XRAY_ASSET}" -o "$tmp"
  local d=/tmp/xray-extract
  rm -rf "$d" && mkdir "$d" && unzip -o "$tmp" -d "$d"
  install -m 755 "$d/xray" "$INSTALL_DIR/bin/xray"
  cp -f "$d/geoip.dat" "$d/geosite.dat" "$INSTALL_DIR/bin/" 2>/dev/null || true
  rm -rf "$tmp" "$d"
  local xver
  xver=$("$INSTALL_DIR/bin/xray" version 2>/dev/null || true)
  printf '%s\n' "${xver%%$'\n'*}"
}

rand_str() {
  local chars=abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789
  local s="" i n="${1:-12}"
  for ((i = 0; i < n; i++)); do
    s+=${chars:RANDOM%62:1}
  done
  printf '%s' "$s"
}

public_ip() {
  curl -4 -fsSL --max-time 5 https://ifconfig.me/ip 2>/dev/null \
    || curl -4 -fsSL --max-time 5 https://api.ipify.org 2>/dev/null \
    || hostname -I 2>/dev/null | awk '{print $1}' \
    || echo ""
}

enable_bbr() {
  if sysctl net.ipv4.tcp_congestion_control 2>/dev/null | grep -q bbr; then
    return
  fi
  log "включаю BBR"
  cat >/etc/sysctl.d/99-panelvpn-bbr.conf <<'EOF'
net.core.default_qdisc=fq
net.ipv4.tcp_congestion_control=bbr
EOF
  sysctl --system >/dev/null 2>&1 || true
}

open_port() {
  local p="$1"
  if command -v ufw >/dev/null 2>&1; then
    ufw allow "${p}/tcp" comment panelvpn >/dev/null 2>&1 || true
  fi
  if command -v firewall-cmd >/dev/null 2>&1; then
    firewall-cmd --permanent --add-port="${p}/tcp" >/dev/null 2>&1 || true
    firewall-cmd --reload >/dev/null 2>&1 || true
  fi
}

install_pkgs
download_panel
install_xray

set +e
set +u
set +o pipefail

HOST="${PANEL_HOST:-$(public_ip)}"
USERNAME="${PANEL_USERNAME:-$(rand_str 8)}"
PASSWORD="${PANEL_PASSWORD:-$(rand_str 12)}"
PORT="${PANEL_PORT:-2053}"
VPN_PORT="${PANEL_VPN_PORT:-443}"
CREATE_INBOUND="${CREATE_INBOUND:-Y}"

log "логин и пароль сгенерированы, вопросов не будет"

cat >"$ENV_DIR/panel.env" <<EOF
PANEL_LISTEN=0.0.0.0:${PORT}
PANEL_DATA=${INSTALL_DIR}/data
XRAY_BIN=${INSTALL_DIR}/bin/xray
PANEL_ENV=${ENV_DIR}/panel.env
EOF
chmod 600 "$ENV_DIR/panel.env"

"$INSTALL_DIR/panel" setting -username "$USERNAME" -password "$PASSWORD" -port "$PORT" -host "$HOST"

LINK=""
if [[ "$CREATE_INBOUND" =~ ^[YyДд] ]]; then
  LINK="$("$INSTALL_DIR/panel" init-inbound -port "$VPN_PORT" -name main -host "$HOST" | tail -n1)"
  open_port "$VPN_PORT"
fi
open_port "$PORT"

enable_bbr

systemctl daemon-reload
systemctl enable panelvpn
systemctl restart panelvpn
sleep 1
systemctl --no-pager --full status panelvpn | head -n 15 || true

URL="http://${HOST:-SERVER_IP}:${PORT}"
umask 077
cat >"$ENV_DIR/install-result.env" <<EOF
PANEL_USERNAME=$(printf '%q' "$USERNAME")
PANEL_PASSWORD=$(printf '%q' "$PASSWORD")
PANEL_PORT=$(printf '%q' "$PORT")
PANEL_URL=$(printf '%q' "$URL")
PANEL_HOST=$(printf '%q' "$HOST")
PANEL_VPN_PORT=$(printf '%q' "$VPN_PORT")
EOF
chmod 600 "$ENV_DIR/install-result.env"

echo
echo -e "${green}════════════════════════════════════${plain}"
echo -e "${green}  Kite установлен${plain}"
echo -e "${green}════════════════════════════════════${plain}"
echo -e "Панель:   ${yellow}${URL}${plain}"
echo -e "Логин:    ${yellow}${USERNAME}${plain}"
echo -e "Пароль:   ${yellow}${PASSWORD}${plain}"
if [[ -n "$LINK" ]]; then
  echo -e "VPN ключ: ${yellow}${LINK}${plain}"
  echo "Импортируй в v2rayN / Streisand / Happ / Hiddify"
fi
echo
echo "Потом сменишь в панели → Настройки, или: panelvpn"
echo "Доступ:      cat /etc/panelvpn/install-result.env"
echo "Если с другой сети не открывается — открой порт ${PORT} в файрволе провайдера."
echo
