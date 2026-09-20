#!/usr/bin/env bash
# Kite — установка на VPS, как 3X-UI:
#
#   bash <(curl -Ls https://raw.githubusercontent.com/remnaweb/kite/main/install.sh)
#
set -euo pipefail

red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; plain='\033[0m'
INSTALL_DIR=/usr/local/panelvpn
ENV_DIR=/etc/panelvpn
XRAY_VERSION="${XRAY_VERSION:-v25.8.3}"
GO_VERSION="${GO_VERSION:-1.22.10}"
NODE_VERSION="${NODE_VERSION:-v22.12.0}"
KITE_REPO="${KITE_REPO:-https://github.com/remnaweb/kite.git}"
KITE_BRANCH="${KITE_BRANCH:-main}"

log()  { echo -e "${green}[Kite]${plain} $*"; }
warn() { echo -e "${yellow}[Kite]${plain} $*"; }
die()  { echo -e "${red}[Kite]${plain} $*" >&2; exit 1; }

[[ $EUID -eq 0 ]] || die "Нужен root: sudo bash install.sh"
[[ "$(uname -s)" == Linux ]] || die "Скрипт для Linux VPS."

avail="$(df -Pm / | awk 'NR==2{print $4}')"
if [[ "${avail:-0}" -lt 1500 ]]; then
  warn "мало места: ${avail:-?} MB свободно. Чищу кэш…"
  apt-get clean >/dev/null 2>&1 || true
  rm -rf /root/go/pkg/mod /root/go/pkg/mod/cache /tmp/go* /tmp/node* /tmp/xray* /tmp/*.zip /tmp/*.tgz
  avail="$(df -Pm / | awk 'NR==2{print $4}')"
  [[ "${avail:-0}" -lt 800 ]] && die "на диске ${avail:-0} MB. Нужно хотя бы ~1.5 GB. Расширь диск или удали лишнее: df -h"
fi

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) GOARCH=amd64; NODE_ARCH=x64; XRAY_ASSET=Xray-linux-64.zip ;;
  aarch64|arm64) GOARCH=arm64; NODE_ARCH=arm64; XRAY_ASSET=Xray-linux-arm64-v8a.zip ;;
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
      apt-get install -y curl wget tar unzip ca-certificates git xz-utils build-essential
      ;;
    centos|rhel|rocky|almalinux|fedora)
      (yum install -y curl wget tar unzip ca-certificates git xz gcc make || dnf install -y curl wget tar unzip ca-certificates git xz gcc make)
      ;;
    *)
      warn "неизвестный дистрибутив ${ID:-}, ставлю без пакетного менеджера"
      ;;
  esac
}

fetch_src() {
  local here=""
  if [[ -n "${BASH_SOURCE[0]:-}" && -f "${BASH_SOURCE[0]}" ]]; then
    here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)" || here=""
    if [[ -n "$here" && -f "$here/go.mod" ]]; then
      SRC="$here"
      log "исходники: $SRC"
      return
    fi
  fi
  log "скачиваю Kite из $KITE_REPO …"
  command -v git >/dev/null 2>&1 || die "нет git"
  SRC=/usr/local/kite-src
  rm -rf "$SRC"
  if ! git clone --depth 1 -b "$KITE_BRANCH" "$KITE_REPO" "$SRC"; then
    die "не удалось скачать репозиторий. Он должен быть публичным: $KITE_REPO"
  fi
  [[ -f "$SRC/go.mod" ]] || die "в репозитории нет go.mod"
}

have_go() {
  command -v go >/dev/null 2>&1 && go version | grep -qE 'go1\.(2[2-9]|[3-9][0-9])'
}

install_go() {
  have_go && { log "Go уже есть: $(go version)"; return; }
  log "ставлю Go ${GO_VERSION}"
  local tmp="/tmp/go${GO_VERSION}.tgz"
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz" -o "$tmp"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "$tmp"
  rm -f "$tmp"
  export PATH="/usr/local/go/bin:$PATH"
  grep -q '/usr/local/go/bin' /etc/profile || echo 'export PATH=/usr/local/go/bin:$PATH' >> /etc/profile
}

install_node() {
  command -v npm >/dev/null 2>&1 && { log "npm уже есть"; return; }
  log "ставлю Node ${NODE_VERSION}"
  local name="node-${NODE_VERSION}-linux-${NODE_ARCH}"
  local tmp="/tmp/${name}.tar.xz"
  curl -fsSL "https://nodejs.org/dist/${NODE_VERSION}/${name}.tar.xz" -o "$tmp"
  tar -C /usr/local -xJf "$tmp"
  ln -sfn "/usr/local/${name}/bin/node" /usr/local/bin/node
  ln -sfn "/usr/local/${name}/bin/npm" /usr/local/bin/npm
  rm -f "$tmp"
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
  "$INSTALL_DIR/bin/xray" version | head -n1
}

build_panel() {
  log "собираю панель…"
  export PATH="/usr/local/go/bin:/usr/local/bin:$PATH"
  export GOTOOLCHAIN=local
  export CGO_ENABLED=1
  cd "$SRC/web"
  npm install --no-audit --no-fund
  npm run build
  rm -rf node_modules
  cd "$SRC"
  mkdir -p "$INSTALL_DIR/web"
  rm -rf "$INSTALL_DIR/web/dist"
  cp -a web/dist "$INSTALL_DIR/web/dist"
  go build -o "$INSTALL_DIR/panel" ./cmd/panel
  chmod 755 "$INSTALL_DIR/panel"
  go clean -modcache >/dev/null 2>&1 || true
}

rand_str() { tr -dc 'A-Za-z0-9' </dev/urandom | head -c "${1:-12}"; }

public_ip() {
  curl -4 -fsSL --max-time 5 https://ifconfig.me/ip 2>/dev/null \
    || curl -4 -fsSL --max-time 5 https://api.ipify.org 2>/dev/null \
    || hostname -I 2>/dev/null | awk '{print $1}' \
    || echo ""
}

prompt() {
  local var="$1" msg="$2" def="${3:-}"
  local val=""
  if [[ "${PANEL_NONINTERACTIVE:-0}" == "1" ]] || [[ ! -t 0 ]]; then
    eval "val=\${$var:-}"
    [[ -z "$val" ]] && val="$def"
  else
    if [[ -n "$def" ]]; then
      read -rp "$msg [$def]: " val || true
      val=${val:-$def}
    else
      read -rp "$msg: " val || true
    fi
  fi
  eval "$var=\"\$val\""
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
fetch_src
install_go
install_node
mkdir -p "$INSTALL_DIR" "$INSTALL_DIR/data" "$ENV_DIR"
install_xray
build_panel
cp -f "$SRC/deploy/panelvpn.service" /etc/systemd/system/panelvpn.service
install -m 755 "$SRC/scripts/panelvpn.sh" /usr/local/bin/panelvpn

HOST="$(public_ip)"
echo
echo -e "${green}════ данные админки ════${plain}"
USERNAME="${PANEL_USERNAME:-}"
PASSWORD="${PANEL_PASSWORD:-}"
PORT="${PANEL_PORT:-}"
VPN_PORT="${PANEL_VPN_PORT:-}"
prompt USERNAME "Логин панели" "admin"
if [[ -z "${PASSWORD}" && ( "${PANEL_NONINTERACTIVE:-0}" == "1" || ! -t 0 ) ]]; then
  PASSWORD="$(rand_str 12)"
fi
if [[ -z "${PASSWORD}" ]]; then
  gen="$(rand_str 12)"
  prompt PASSWORD "Пароль панели (пусто = случайный ${gen})" "$gen"
fi
prompt PORT "Порт панели" "2053"
prompt HOST "Публичный IP/домен для ключей" "$HOST"
prompt VPN_PORT "Порт VPN (VLESS Reality)" "443"
CREATE_INBOUND="${CREATE_INBOUND:-Y}"
if [[ "${PANEL_NONINTERACTIVE:-0}" != "1" && -t 0 ]]; then
  read -rp "Создать рабочий инбаунд VLESS Reality сейчас? [Y/n]: " CREATE_INBOUND || true
  CREATE_INBOUND=${CREATE_INBOUND:-Y}
fi

export PANEL_DATA="$INSTALL_DIR/data"
export XRAY_BIN="$INSTALL_DIR/bin/xray"
export PANEL_WEB="$INSTALL_DIR/web/dist"
export PANEL_ENV="$ENV_DIR/panel.env"
export PANEL_LISTEN="0.0.0.0:${PORT}"

cat >"$ENV_DIR/panel.env" <<EOF
PANEL_LISTEN=0.0.0.0:${PORT}
PANEL_DATA=${INSTALL_DIR}/data
XRAY_BIN=${INSTALL_DIR}/bin/xray
PANEL_WEB=${INSTALL_DIR}/web/dist
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
echo "Управление:  panelvpn"
echo "Доступ:      cat /etc/panelvpn/install-result.env"
echo "Если с другой сети не открывается — открой порт ${PORT} в файрволе провайдера."
echo
