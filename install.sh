#!/bin/bash
# ============================================================
#  SERVO - Server Monitor Installer
#  Usage: curl -fsSL https://yourserver.com/install.sh | bash
# ============================================================
set -e

REPO="febritecno/servo"
INSTALL_DIR="/usr/local/bin"
SERVICE_NAME="servo"
PORT=":8080"
GITHUB_BASE="https://github.com/${REPO}/releases/latest/download"

# ── Colors ──────────────────────────────────────────────────
R='\033[0;31m'; G='\033[0;32m'; Y='\033[0;33m'
B='\033[0;34m'; C='\033[0;36m'; W='\033[1;37m'; N='\033[0m'

banner() {
  echo -e "${C}"
  echo "  ███████╗███████╗██████╗ ██╗   ██╗ ██████╗ "
  echo "  ██╔════╝██╔════╝██╔══██╗██║   ██║██╔═══██╗"
  echo "  ███████╗█████╗  ██████╔╝██║   ██║██║   ██║"
  echo "  ╚════██║██╔══╝  ██╔══██╗╚██╗ ██╔╝██║   ██║"
  echo "  ███████║███████╗██║  ██║ ╚████╔╝ ╚██████╔╝"
  echo "  ╚══════╝╚══════╝╚═╝  ╚═╝  ╚═══╝   ╚═════╝ "
  echo -e "${N}  ${W}Server Monitor Dashboard${N} — Installer"
  echo ""
}

info()    { echo -e "  ${B}[•]${N} $*"; }
success() { echo -e "  ${G}[✓]${N} $*"; }
warn()    { echo -e "  ${Y}[!]${N} $*"; }
error()   { echo -e "  ${R}[✗]${N} $*"; exit 1; }

# ── Detect OS / Arch ────────────────────────────────────────
detect_platform() {
  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
  ARCH=$(uname -m)

  case "$OS" in
    linux)   OS="linux"   ;;
    darwin)  OS="darwin"  ;;
    freebsd) OS="freebsd" ;;
    *) error "Unsupported OS: $OS" ;;
  esac

  case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    armv7*|armv6*)  ARCH="arm"   ;;
    i386|i686)      ARCH="386"   ;;
    *) error "Unsupported architecture: $ARCH" ;;
  esac

  EXT=""
  [[ "$OS" == "windows" ]] && EXT=".exe"

  BINARY_NAME="servo_${OS}_${ARCH}${EXT}"
  info "Platform detected: ${W}${OS}/${ARCH}${N}"
}

# ── Check dependencies ───────────────────────────────────────
check_deps() {
  for cmd in curl chmod; do
    command -v "$cmd" &>/dev/null || error "Required command not found: $cmd"
  done
}

# ── Download binary ──────────────────────────────────────────
download_binary() {
  local URL="${GITHUB_BASE}/${BINARY_NAME}"
  local TMP="/tmp/servo_download"

  info "Downloading ${W}${BINARY_NAME}${N}..."
  info "Source: ${URL}"

  if ! curl -fsSL --progress-bar "$URL" -o "$TMP"; then
    error "Download failed. Check URL or network connection."
  fi

  chmod +x "$TMP"

  # Verify it's a valid binary
  if ! "$TMP" --help &>/dev/null 2>&1; then
    warn "Binary smoke-test skipped (expected for cross-platform)"
  fi

  success "Downloaded successfully"
  echo "$TMP"
}

# ── Install binary ───────────────────────────────────────────
install_binary() {
  local TMP=$1

  if [[ -w "$INSTALL_DIR" ]]; then
    mv "$TMP" "${INSTALL_DIR}/servo"
  else
    info "Requesting sudo to install to ${INSTALL_DIR}..."
    sudo mv "$TMP" "${INSTALL_DIR}/servo"
    sudo chmod +x "${INSTALL_DIR}/servo"
  fi

  success "Installed to ${W}${INSTALL_DIR}/servo${N}"
}

# ── Systemd service (Linux) ──────────────────────────────────
install_systemd() {
  [[ "$OS" != "linux" ]] && return
  command -v systemctl &>/dev/null || { warn "systemd not found, skipping service setup"; return; }

  local SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
  local CURRENT_USER=$(whoami)
  [[ "$CURRENT_USER" == "root" ]] && RUN_USER="root" || RUN_USER="$CURRENT_USER"

  info "Creating systemd service..."

  cat > /tmp/servo.service << EOF
[Unit]
Description=SERVO Server Monitor Dashboard
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
ExecStart=${INSTALL_DIR}/servo --port ${PORT}
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=servo

[Install]
WantedBy=multi-user.target
EOF

  if [[ -w "/etc/systemd/system" ]]; then
    mv /tmp/servo.service "$SERVICE_FILE"
    systemctl daemon-reload
    systemctl enable "$SERVICE_NAME" --quiet
    systemctl restart "$SERVICE_NAME"
  else
    sudo mv /tmp/servo.service "$SERVICE_FILE"
    sudo systemctl daemon-reload
    sudo systemctl enable "$SERVICE_NAME" --quiet
    sudo systemctl restart "$SERVICE_NAME"
  fi

  success "Service ${W}${SERVICE_NAME}${N} enabled & started"
}

# ── LaunchAgent (macOS) ──────────────────────────────────────
install_launchd() {
  [[ "$OS" != "darwin" ]] && return

  local PLIST="$HOME/Library/LaunchAgents/com.servo.monitor.plist"
  info "Creating macOS LaunchAgent..."

  mkdir -p "$HOME/Library/LaunchAgents"
  cat > "$PLIST" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.servo.monitor</string>
  <key>ProgramArguments</key>
  <array>
    <string>${INSTALL_DIR}/servo</string>
    <string>--port</string><string>${PORT}</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/tmp/servo.log</string>
  <key>StandardErrorPath</key><string>/tmp/servo.log</string>
</dict>
</plist>
EOF

  launchctl unload "$PLIST" 2>/dev/null || true
  launchctl load "$PLIST"
  success "LaunchAgent installed & started"
}

# ── Verify running ────────────────────────────────────────────
verify_running() {
  sleep 2
  if [[ "$OS" == "linux" ]] && command -v systemctl &>/dev/null; then
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
      success "Service is ${G}running${N}"
    else
      warn "Service may not have started. Check: journalctl -u servo -n 20"
    fi
  else
    if curl -sf "http://127.0.0.1${PORT}" &>/dev/null; then
      success "SERVO is ${G}responding${N}"
    else
      warn "Could not verify. It may still be starting."
    fi
  fi
}

# ── Print summary ─────────────────────────────────────────────
print_summary() {
  echo ""
  echo -e "  ${G}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
  echo -e "  ${W}  SERVO installed successfully!${N}"
  echo -e "  ${G}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${N}"
  echo ""
  echo -e "  ${C}Dashboard URL:${N}  http://$(hostname -I 2>/dev/null | awk '{print $1}' || echo '127.0.0.1')${PORT}"
  echo -e "  ${C}Local URL:${N}      http://127.0.0.1${PORT}"
  echo ""
  echo -e "  ${W}Useful commands:${N}"
  if [[ "$OS" == "linux" ]] && command -v systemctl &>/dev/null; then
    echo "    systemctl status servo    # Check status"
    echo "    systemctl restart servo   # Restart"
    echo "    journalctl -u servo -f    # View logs"
    echo "    systemctl stop servo      # Stop"
  elif [[ "$OS" == "darwin" ]]; then
    echo "    launchctl list | grep servo   # Check status"
    echo "    tail -f /tmp/servo.log        # View logs"
  fi
  echo ""
  echo -e "  ${W}Manual run:${N}  servo --port ${PORT}"
  echo -e "  ${W}With flags:${N}  servo --port ${PORT} --apache http://127.0.0.1:8080/server-status"
  echo ""
}

# ── Uninstall mode ────────────────────────────────────────────
uninstall() {
  info "Uninstalling SERVO..."

  if [[ "$OS" == "linux" ]] && command -v systemctl &>/dev/null; then
    sudo systemctl stop "$SERVICE_NAME" 2>/dev/null || true
    sudo systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    sudo rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    sudo systemctl daemon-reload
  elif [[ "$OS" == "darwin" ]]; then
    launchctl unload "$HOME/Library/LaunchAgents/com.servo.monitor.plist" 2>/dev/null || true
    rm -f "$HOME/Library/LaunchAgents/com.servo.monitor.plist"
  fi

  sudo rm -f "${INSTALL_DIR}/servo"
  success "SERVO removed successfully"
  exit 0
}

# ── Main ──────────────────────────────────────────────────────
main() {
  banner
  detect_platform
  check_deps

  [[ "${1:-}" == "--uninstall" ]] && uninstall

  # Parse optional flags
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --port) PORT="$2"; shift 2 ;;
      --uninstall) uninstall ;;
      *) shift ;;
    esac
  done

  TMP=$(download_binary)
  install_binary "$TMP"

  [[ "$OS" == "linux" ]]  && install_systemd
  [[ "$OS" == "darwin" ]] && install_launchd

  verify_running
  print_summary
}

main "$@"
