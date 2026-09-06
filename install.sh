#!/bin/bash

# MKGames Server Installer
# Futuristic installer with progress bar

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'
DIM='\033[2m'

INSTALL_DIR="/opt/mkgames"
DATA_DIR="/var/lib/mkgames"
LOG_DIR="/var/log/mkgames"
CONFIG_DIR="/etc/mkgames"
SERVICE_FILE="/etc/systemd/system/mkgames.service"
ADMIN_PASSCODE=""

TOTAL_STEPS=9
CURRENT_STEP=0

print_banner() {
    clear
    echo -e "${GREEN}"
    echo "╔══════════════════════════════════════════════════╗"
    echo "║                                                  ║"
    echo "║   ███╗   ███╗ ███╗   ███╗ ██████╗               ║"
    echo "║   ████╗ ████║ ████╗ ████║ ██╔══██╗              ║"
    echo "║   ██╔████╔██║ ██╔████╔██║ ██║  ██║              ║"
    echo "║   ██║╚██╔╝██║ ██║╚██╔╝██║ ██║  ██║              ║"
    echo "║   ██║ ╚═╝ ██║ ██║ ╚═╝ ██║ ██████╔╝              ║"
    echo "║   ╚═╝     ╚═╝ ╚═╝     ╚═╝ ╚═════╝              ║"
    echo "║                                                  ║"
    echo "║          SERVER INSTALLER v1.0.0                  ║"
    echo "║                                                  ║"
    echo "╚══════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

progress_bar() {
    local step=$1
    local total=$2
    local desc=$3
    local pct=$((step * 100 / total))
    local filled=$((pct / 2))
    local empty=$((50 - filled))

    local bar=""
    for ((i=0; i<filled; i++)); do bar+="█"; done
    for ((i=0; i<empty; i++)); do bar+="░"; done

    echo ""
    echo -e "  ${CYAN}[$bar]${NC} ${BOLD}${pct}%${NC}"
    echo -e "  ${DIM}${desc}${NC}"
    echo ""
}

step_done() {
    echo -e "  ${GREEN}✓${NC} $1"
}

step_info() {
    echo -e "  ${CYAN}→${NC} $1"
}

step_warn() {
    echo -e "  ${YELLOW}!${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        echo -e "${RED}This installer must be run as root${NC}"
        echo "Usage: sudo ./install.sh"
        exit 1
    fi
}

configure_admin_passcode() {
    if [[ -f "$DATA_DIR/mkgames.db" ]] && command -v sqlite3 >/dev/null 2>&1; then
        EXISTING_ADMINS=$(sqlite3 "$DATA_DIR/mkgames.db" "SELECT COUNT(*) FROM admins;" 2>/dev/null || echo 0)
        if [[ "$EXISTING_ADMINS" =~ ^[1-9][0-9]*$ ]]; then
            echo -e "${YELLOW}Existing admin database found; keeping the current admin passcode${NC}"
            return
        fi
    fi

    while [[ -z "$ADMIN_PASSCODE" ]]; do
        read -r -s -p "Choose the initial admin passcode: " ADMIN_PASSCODE
        echo
        if [[ -z "$ADMIN_PASSCODE" ]]; then
            echo -e "${RED}Passcode cannot be empty${NC}"
            continue
        fi
        if [[ ! "$ADMIN_PASSCODE" =~ ^[a-zA-Z0-9._-]+$ ]]; then
            echo -e "${RED}Use only letters, numbers, dot, underscore, or hyphen${NC}"
            ADMIN_PASSCODE=""
        fi
    done
}

detect_os() {
    if [[ ! -f /etc/os-release ]]; then
        echo -e "${RED}Cannot detect OS. This installer supports Ubuntu/Debian only${NC}"
        exit 1
    fi

    . /etc/os-release
    if [[ "$ID" != "ubuntu" && "$ID" != "debian" ]]; then
        step_warn "Detected $ID. This installer is designed for Ubuntu. Proceeding anyway..."
    fi
    step_done "OS detected: $PRETTY_NAME"
}

install_dependencies() {
    progress_bar 1 $TOTAL_STEPS "Installing system dependencies..."

    step_info "Updating package lists..."
    apt-get update -qq

    step_info "Installing build tools..."
    apt-get install -y -qq curl wget git build-essential ca-certificates
    step_done "Build tools installed"

    step_info "Installing Go..."
    if command -v go &>/dev/null; then
        step_done "Go already installed: $(go version)"
    else
        GO_VERSION="1.21.6"
        wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
        rm -rf /usr/local/go
        tar -C /usr/local -xzf /tmp/go.tar.gz
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/golang.sh
        export PATH=$PATH:/usr/local/go/bin
        rm /tmp/go.tar.gz
        step_done "Go ${GO_VERSION} installed"
    fi

    step_info "Installing SQLite..."
    apt-get install -y -qq libsqlite3-dev sqlite3
    step_done "SQLite installed"

    step_info "Installing archive tools..."
    apt-get install -y -qq unzip unrar p7zip-full
    step_done "Archive tools installed"

    step_info "Installing UFW firewall..."
    apt-get install -y -qq ufw
    step_done "UFW installed"
}

setup_directories() {
    progress_bar 2 $TOTAL_STEPS "Creating directories..."

    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$DATA_DIR"
    mkdir -p "$LOG_DIR"
    mkdir -p "$DATA_DIR/games"
    mkdir -p "$DATA_DIR/archives"
    mkdir -p "$DATA_DIR/covers"
    mkdir -p "$DATA_DIR/storage/games" "$DATA_DIR/storage/archives" "$DATA_DIR/storage/covers"

    # Preserve uploads from older installs before replacing the app storage path.
    if [[ -d "$INSTALL_DIR/storage" && ! -L "$INSTALL_DIR/storage" ]]; then
        cp -a "$INSTALL_DIR/storage/." "$DATA_DIR/storage/"
    fi

    # The server stores uploaded files relative to its working directory.
    rm -rf "$INSTALL_DIR/storage"
    ln -s "$DATA_DIR/storage" "$INSTALL_DIR/storage"

    step_done "Application directory: $INSTALL_DIR"
    step_done "Data directory: $DATA_DIR"
    step_done "Log directory: $LOG_DIR"
}

build_server() {
    progress_bar 3 $TOTAL_STEPS "Building server..."

    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SERVER_DIR="$SCRIPT_DIR/server"

    if [[ ! -d "$SERVER_DIR" ]]; then
        echo -e "${RED}Server source not found at $SERVER_DIR${NC}"
        exit 1
    fi

    cd "$SERVER_DIR"

    rm -f "$INSTALL_DIR/mkgames-server.new"
    rm -rf "$INSTALL_DIR/admin-web.new"

    step_info "Downloading Go dependencies..."
    export PATH=$PATH:/usr/local/go/bin
    go mod download
    step_done "Dependencies downloaded"

    step_info "Compiling server binary..."
    go build -trimpath -o "$INSTALL_DIR/mkgames-server.new" .
    chmod 0755 "$INSTALL_DIR/mkgames-server.new"
    mv -f "$INSTALL_DIR/mkgames-server.new" "$INSTALL_DIR/mkgames-server"
    step_done "Server built: $INSTALL_DIR/mkgames-server"

    cp -a "$SCRIPT_DIR/admin/web" "$INSTALL_DIR/admin-web.new"
    rm -rf "$INSTALL_DIR/admin-web"
    mv "$INSTALL_DIR/admin-web.new" "$INSTALL_DIR/admin-web"
    if [[ -n "$ADMIN_PASSCODE" ]]; then
        printf 'MKGAMES_ADMIN_PASSCODE=%s\n' "$ADMIN_PASSCODE" > "$CONFIG_DIR/server.env"
        chmod 600 "$CONFIG_DIR/server.env"
    fi
    step_done "Admin assets copied"

    cd "$SCRIPT_DIR"
}

setup_service() {
    progress_bar 4 $TOTAL_STEPS "Configuring systemd service..."

    cat > "$SERVICE_FILE" << EOF
[Unit]
Description=MKGames Server
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$CONFIG_DIR/server.env
Environment=MKGAMES_ADMIN_WEB_DIR=$INSTALL_DIR/admin-web
ExecStart=$INSTALL_DIR/mkgames-server
Restart=always
RestartSec=5
StandardOutput=append:$LOG_DIR/server.log
StandardError=append:$LOG_DIR/server.log

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable mkgames.service
    if ! systemctl cat mkgames.service >/dev/null 2>&1; then
        echo -e "${RED}Systemd unit was not created correctly${NC}"
        exit 1
    fi
    step_done "Systemd service created and enabled"
}

setup_firewall() {
    progress_bar 5 $TOTAL_STEPS "Configuring firewall..."

    SERVER_PORT="8080"
    if [[ -f "$DATA_DIR/mkgames.db" ]] && command -v sqlite3 >/dev/null 2>&1; then
        CONFIGURED_PORT=$(sqlite3 "$DATA_DIR/mkgames.db" "SELECT wan_port FROM server_config WHERE id=1;" 2>/dev/null || true)
        if [[ "$CONFIGURED_PORT" =~ ^[0-9]+$ ]] && (( CONFIGURED_PORT >= 1 && CONFIGURED_PORT <= 65535 )); then
            SERVER_PORT="$CONFIGURED_PORT"
        fi
    fi

    step_info "Opening server port $SERVER_PORT..."
    ufw allow "$SERVER_PORT/tcp"
    step_done "Port $SERVER_PORT opened"

    step_info "Allowing SSH before enabling firewall..."
    ufw allow OpenSSH || ufw allow 22/tcp
    step_done "SSH access allowed"

    step_info "Enabling UFW..."
    ufw --force enable
    step_done "Firewall enabled"
}

create_cli() {
    progress_bar 6 $TOTAL_STEPS "Installing CLI tool..."

    cat > /usr/local/bin/mkgames << 'CLIEOF'
#!/bin/bash
exec /opt/mkgames/mkgames-server "$@"
CLIEOF

    chmod +x /usr/local/bin/mkgames
    ln -sf /usr/local/bin/mkgames /usr/local/bin/mklauncher
    step_done "CLI tools installed: /usr/local/bin/mkgames and /usr/local/bin/mklauncher"
}

start_service() {
    progress_bar 8 $TOTAL_STEPS "Starting server..."

    systemctl daemon-reload
    systemctl enable mkgames.service
    if ! systemctl restart mkgames.service; then
        echo -e "${RED}Server failed to start. Recent logs:${NC}"
        journalctl -u mkgames.service -n 40 --no-pager || true
        exit 1
    fi
    systemctl is-active --quiet mkgames.service
    step_done "Server started and enabled"
}

setup_logrotate() {
    progress_bar 7 $TOTAL_STEPS "Configuring log rotation..."

    cat > /etc/logrotate.d/mkgames << EOF
$LOG_DIR/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0644 root root
}
EOF

    step_done "Log rotation configured"
}

print_complete() {
    progress_bar 9 $TOTAL_STEPS "Installation complete!"

    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║          INSTALLATION COMPLETE!                  ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  ${BOLD}Quick Start:${NC}"
    echo -e "  ${CYAN}1.${NC} Start server:    ${BOLD}systemctl start mkgames${NC}"
    echo -e "  ${CYAN}2.${NC} Stop server:     ${BOLD}systemctl stop mkgames${NC}"
    echo -e "  ${CYAN}3.${NC} View status:     ${BOLD}mkgames status${NC}"
    echo -e "  ${CYAN}4.${NC} Show WAN info:   ${BOLD}mkgames wan${NC}"
    echo -e "  ${CYAN}5.${NC} Admin panel:     ${BOLD}http://YOUR_IP:8080${NC}"
    echo ""
    echo -e "  ${DIM}Admin passcode: chosen during installation${NC}"
    echo -e "  ${DIM}Logs: $LOG_DIR/server.log${NC}"
    echo ""
    echo -e "  ${YELLOW}Server is running. Use 'systemctl status mkgames' to check it${NC}"
    echo ""
}

main() {
    print_banner
    check_root
    detect_os
    echo ""
    install_dependencies
    setup_directories
    configure_admin_passcode
    build_server
    setup_service
    setup_firewall
    create_cli
    setup_logrotate
    start_service
    print_complete
}

main "$@"
