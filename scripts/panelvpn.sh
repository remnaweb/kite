#!/usr/bin/env bash
# Меню управления panel.vpn (как x-ui)
set -euo pipefail
red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; plain='\033[0m'
BIN=/usr/local/panelvpn/panel

need_root() {
  if [[ $EUID -ne 0 ]]; then
    echo -e "${red}Запусти от root: sudo panelvpn${plain}"
    exit 1
  fi
}

status() { systemctl status panelvpn --no-pager || true; }

show_creds() {
  echo
  if [[ -f /etc/panelvpn/install-result.env ]]; then
    # shellcheck disable=SC1091
    source /etc/panelvpn/install-result.env
    echo -e "URL:      ${green}${PANEL_URL}${plain}"
    echo -e "Логин:    ${green}${PANEL_USERNAME}${plain}"
    echo -e "Пароль:   ${green}${PANEL_PASSWORD}${plain}"
  fi
  "$BIN" setting || true
  echo
}

reset_admin() {
  read -rp "Новый логин [admin]: " u; u=${u:-admin}
  read -rp "Новый пароль: " p
  [[ -z "$p" ]] && { echo "пароль пустой"; return; }
  read -rp "Порт панели [2053]: " port; port=${port:-2053}
  read -rp "Публичный IP/домен (для ссылок): " host
  "$BIN" setting -username "$u" -password "$p" -port "$port" ${host:+-host "$host"}
  systemctl restart panelvpn
  echo -e "${green}Готово. Перезайди в панель.${plain}"
}

case "${1:-}" in
  start)   need_root; systemctl start panelvpn; status ;;
  stop)    need_root; systemctl stop panelvpn; status ;;
  restart) need_root; systemctl restart panelvpn; status ;;
  status)  status ;;
  log)     journalctl -u panelvpn -e --no-pager ;;
  setting) shift; need_root; "$BIN" setting "$@"; [[ $# -gt 0 ]] && systemctl restart panelvpn ;;
  *)
    need_root
    while true; do
      echo
      echo -e "${green}Kite${plain}"
      echo "  0) выход"
      echo "  1) старт"
      echo "  2) стоп"
      echo "  3) рестарт"
      echo "  4) статус"
      echo "  5) логи"
      echo "  6) показать доступ"
      echo "  7) сменить логин/пароль/порт"
      read -rp "пункт: " n
      case "$n" in
        0) exit 0 ;;
        1) systemctl start panelvpn ;;
        2) systemctl stop panelvpn ;;
        3) systemctl restart panelvpn ;;
        4) status ;;
        5) journalctl -u panelvpn -n 80 --no-pager ;;
        6) show_creds ;;
        7) reset_admin ;;
        *) echo "нет такого пункта" ;;
      esac
    done
    ;;
esac
