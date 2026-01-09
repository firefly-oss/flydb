#!/bin/bash
#
# FlyDB Installation Script
# Copyright (c) 2026 Firefly Software Solutions Inc.
# Licensed under the Apache License, Version 2.0
#
# A best-in-class installation experience with both interactive wizard
# and non-interactive CLI modes. Supports both local source builds and
# remote installation via pre-built binaries.
#
# Usage:
#   Remote install:  curl -sSL https://get.flydb.dev | bash
#   With options:    curl -sSL https://get.flydb.dev | bash -s -- --prefix ~/.local --yes
#   Interactive:     ./install.sh
#   Non-interactive: ./install.sh --prefix /usr/local --yes
#   From source:     ./install.sh --from-source
#   Uninstall:       ./install.sh --uninstall
#

set -euo pipefail

# =============================================================================
# Configuration and Defaults
# =============================================================================

readonly SCRIPT_VERSION="01.26.9"
readonly FLYDB_VERSION="${FLYDB_VERSION:-01.26.9}"
readonly GITHUB_REPO="firefly-software/flydb"
readonly DOWNLOAD_BASE_URL="https://github.com/${GITHUB_REPO}/releases/download"

# Default values (can be overridden by CLI args or interactive prompts)
PREFIX=""
INSTALL_SERVICE=true
CREATE_CONFIG=true
INIT_DATABASE=false
AUTO_CONFIRM=false
UNINSTALL=false
SPECIFIC_VERSION=""
INTERACTIVE=true
# Installation mode: "auto" (detect), "source" (build from source), "binary" (download pre-built)
INSTALL_MODE="auto"
# Temporary directory for downloads
TEMP_DIR=""

# Detected system info
OS=""
ARCH=""
DISTRO=""
INIT_SYSTEM=""

# Installation tracking for rollback
declare -a INSTALLED_FILES=()
declare -a CREATED_DIRS=()
INSTALL_STARTED=false

# Resolved installation mode after detection
RESOLVED_INSTALL_MODE=""

# =============================================================================
# Colors and Formatting (matching pkg/cli/colors.go)
# =============================================================================

# Check if colors should be enabled
if [[ -t 1 ]] && [[ -z "${NO_COLOR:-}" ]]; then
    readonly COLOR_ENABLED=true
else
    readonly COLOR_ENABLED=false
fi

# ANSI color codes
if [[ "$COLOR_ENABLED" == true ]]; then
    readonly RESET='\033[0m'
    readonly BOLD='\033[1m'
    readonly DIM='\033[2m'
    readonly RED='\033[31m'
    readonly GREEN='\033[32m'
    readonly YELLOW='\033[33m'
    readonly BLUE='\033[34m'
    readonly MAGENTA='\033[35m'
    readonly CYAN='\033[36m'
    readonly BRIGHT_BLACK='\033[90m'
else
    readonly RESET=''
    readonly BOLD=''
    readonly DIM=''
    readonly RED=''
    readonly GREEN=''
    readonly YELLOW=''
    readonly BLUE=''
    readonly MAGENTA=''
    readonly CYAN=''
    readonly BRIGHT_BLACK=''
fi

# Icons (matching pkg/cli/colors.go)
readonly ICON_SUCCESS="✓"
readonly ICON_ERROR="✗"
readonly ICON_WARNING="⚠"
readonly ICON_INFO="ℹ"
readonly ICON_ARROW="→"

# Spinner frames (matching pkg/cli/spinner.go)
readonly SPINNER_FRAMES=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")

# =============================================================================
# Output Functions
# =============================================================================

print_success() {
    echo -e "${GREEN}${ICON_SUCCESS}${RESET} ${GREEN}$1${RESET}"
}

print_error() {
    echo -e "${RED}${ICON_ERROR}${RESET} ${RED}$1${RESET}" >&2
}

print_warning() {
    echo -e "${YELLOW}${ICON_WARNING}${RESET} ${YELLOW}$1${RESET}"
}

print_info() {
    echo -e "${CYAN}${ICON_INFO}${RESET} ${CYAN}$1${RESET}"
}

print_step() {
    echo -e "${BLUE}${BOLD}==>${RESET} ${BOLD}$1${RESET}"
}

print_substep() {
    echo -e "    ${ICON_ARROW} $1"
}

print_dim() {
    echo -e "${DIM}$1${RESET}"
}

separator() {
    local width="${1:-60}"
    printf '%*s\n' "$width" '' | tr ' ' '─'
}

double_separator() {
    local width="${1:-60}"
    printf '%*s\n' "$width" '' | tr ' ' '═'
}

# Key-value display (matching pkg/cli/output.go KeyValue function)
print_kv() {
    local key="$1"
    local value="$2"
    local width="${3:-22}"
    printf "  %-${width}s %b\n" "${key}:" "$value"
}

# =============================================================================
# Spinner Functions
# =============================================================================

SPINNER_PID=""
SPINNER_ACTIVE=false

spinner_start() {
    local message="$1"

    # Don't start spinner if not interactive terminal
    if [[ ! -t 1 ]]; then
        echo "$message..."
        return
    fi

    # Stop any existing spinner first
    spinner_stop

    SPINNER_ACTIVE=true

    (
        local i=0
        while true; do
            local frame="${SPINNER_FRAMES[$((i % ${#SPINNER_FRAMES[@]}))]}"
            printf "\r${CYAN}%s${RESET} %s" "$frame" "$message"
            sleep 0.08
            ((i++))
        done
    ) &
    SPINNER_PID=$!
    disown "$SPINNER_PID" 2>/dev/null || true
}

spinner_stop() {
    if [[ -n "$SPINNER_PID" ]] && [[ "$SPINNER_ACTIVE" == true ]]; then
        kill "$SPINNER_PID" 2>/dev/null || true
        wait "$SPINNER_PID" 2>/dev/null || true
        SPINNER_PID=""
        SPINNER_ACTIVE=false
        printf "\r\033[K"  # Clear the line
    fi
}

spinner_success() {
    spinner_stop
    print_success "$1"
}

spinner_error() {
    spinner_stop
    print_error "$1"
}

# Ensure spinner is stopped before any interactive prompt
ensure_clean_prompt() {
    spinner_stop
    # Small delay to ensure terminal is ready
    sleep 0.05
}

# =============================================================================
# Banner and Help
# =============================================================================

print_banner() {
    echo ""
    echo -e "${CYAN}${BOLD}╔════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${CYAN}${BOLD}║              FlyDB Installation Script v${SCRIPT_VERSION}            ║${RESET}"
    echo -e "${CYAN}${BOLD}╚════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
}

print_help() {
    echo -e "${BOLD}FlyDB Installation Script${RESET}"
    echo ""
    echo "A best-in-class installation experience for FlyDB - the high-performance"
    echo "embedded SQL database."
    echo ""
    echo -e "${BOLD}USAGE:${RESET}"
    echo "    $0 [OPTIONS]"
    echo ""
    echo "    # Remote installation (download pre-built binaries)"
    echo "    curl -sSL https://get.flydb.dev | bash"
    echo ""
    echo -e "${BOLD}MODES:${RESET}"
    echo "    Interactive (default):  Run without arguments for guided installation"
    echo "    Non-interactive:        Use --yes with other options for scripted installs"
    echo ""
    echo -e "${BOLD}INSTALLATION SOURCE:${RESET}"
    echo "    By default, the script auto-detects whether to build from source or"
    echo "    download pre-built binaries:"
    echo "    - If run from a FlyDB source directory with Go installed: builds from source"
    echo "    - Otherwise: downloads pre-built binaries from GitHub releases"
    echo ""
    echo -e "${BOLD}OPTIONS:${RESET}"
    echo -e "    ${BOLD}--prefix <path>${RESET}"
    echo "        Installation directory for binaries"
    echo "        Default: /usr/local/bin (root) or ~/.local/bin (user)"
    echo ""
    echo -e "    ${BOLD}--version <version>${RESET}"
    echo "        Specific FlyDB version to install"
    echo "        Default: latest (${FLYDB_VERSION})"
    echo ""
    echo -e "    ${BOLD}--from-source${RESET}"
    echo "        Force building from source (requires Go 1.21+)"
    echo "        Must be run from the FlyDB source directory"
    echo ""
    echo -e "    ${BOLD}--from-binary${RESET}"
    echo "        Force downloading pre-built binaries from GitHub"
    echo "        Useful when you want to skip building even in a source directory"
    echo ""
    echo -e "    ${BOLD}--no-service${RESET}"
    echo "        Skip system service installation (systemd/launchd)"
    echo ""
    echo -e "    ${BOLD}--no-config${RESET}"
    echo "        Skip configuration file creation"
    echo ""
    echo -e "    ${BOLD}--init-db${RESET}"
    echo "        Initialize a new database during installation"
    echo ""
    echo -e "    ${BOLD}--yes, -y${RESET}"
    echo "        Skip all confirmation prompts (non-interactive mode)"
    echo ""
    echo -e "    ${BOLD}--uninstall${RESET}"
    echo "        Remove FlyDB installation"
    echo ""
    echo -e "    ${BOLD}--help, -h${RESET}"
    echo "        Show this help message"
    echo ""
    echo -e "${BOLD}EXAMPLES:${RESET}"
    echo "    # Remote installation (recommended for most users)"
    echo "    curl -sSL https://get.flydb.dev | bash"
    echo ""
    echo "    # Remote installation with options"
    echo "    curl -sSL https://get.flydb.dev | bash -s -- --prefix ~/.local --yes"
    echo ""
    echo "    # Interactive installation from source directory"
    echo "    ./install.sh"
    echo ""
    echo "    # Quick install with defaults, no prompts"
    echo "    ./install.sh --yes"
    echo ""
    echo "    # Install to custom location"
    echo "    ./install.sh --prefix /opt/flydb --yes"
    echo ""
    echo "    # Install specific version without service"
    echo "    ./install.sh --version 01.26.0 --no-service --yes"
    echo ""
    echo "    # Force download binaries even in source directory"
    echo "    ./install.sh --from-binary --yes"
    echo ""
    echo "    # User-local installation (no sudo required)"
    echo "    ./install.sh --prefix ~/.local --yes"
    echo ""
    echo "    # Uninstall FlyDB"
    echo "    ./install.sh --uninstall"
    echo ""
    echo -e "${BOLD}ENVIRONMENT VARIABLES:${RESET}"
    echo "    FLYDB_VERSION     Override the default version to install"
    echo "    NO_COLOR          Disable colored output"
    echo ""
    echo -e "${BOLD}SERVER ROLES:${RESET}"
    echo "    standalone        Single server mode (default, no replication)"
    echo "    master            Leader node that accepts writes and replicates to slaves"
    echo "    slave             Follower node that receives replication from master"
    echo "    cluster           Automatic failover cluster with leader election"
    echo ""
    echo -e "${BOLD}CLUSTER CONFIGURATION:${RESET}"
    echo "    After installation, configure cluster mode via:"
    echo "    - Configuration file: /etc/flydb/flydb.conf or ~/.config/flydb/flydb.conf"
    echo "    - Environment variables: FLYDB_ROLE, FLYDB_CLUSTER_PEERS, etc."
    echo "    - Command-line flags: -role cluster -cluster-peers node2:9998,node3:9998"
    echo ""
    echo -e "${BOLD}MORE INFORMATION:${RESET}"
    echo "    Documentation:    https://flydb.dev/docs"
    echo "    GitHub:           https://github.com/${GITHUB_REPO}"
    echo "    Issues:           https://github.com/${GITHUB_REPO}/issues"
    echo ""
}

# =============================================================================
# System Detection
# =============================================================================

detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux)
            OS="linux"
            # Detect Linux distribution
            if [[ -f /etc/os-release ]]; then
                # shellcheck source=/dev/null
                source /etc/os-release
                DISTRO="${ID:-unknown}"
            elif [[ -f /etc/redhat-release ]]; then
                DISTRO="rhel"
            elif [[ -f /etc/debian_version ]]; then
                DISTRO="debian"
            else
                DISTRO="unknown"
            fi
            ;;
        Darwin)
            OS="darwin"
            DISTRO="macos"
            ;;
        MINGW*|MSYS*|CYGWIN*)
            OS="windows"
            DISTRO="windows"
            ;;
        *)
            print_error "Unsupported operating system: $OS"
            exit 1
            ;;
    esac
}

detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv7)
            ARCH="arm"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
}

detect_init_system() {
    if [[ "$OS" == "darwin" ]]; then
        INIT_SYSTEM="launchd"
    elif command -v systemctl &>/dev/null && systemctl --version &>/dev/null; then
        INIT_SYSTEM="systemd"
    elif command -v rc-service &>/dev/null; then
        INIT_SYSTEM="openrc"
    elif [[ -d /etc/init.d ]]; then
        INIT_SYSTEM="sysvinit"
    else
        INIT_SYSTEM="none"
    fi
}

get_default_prefix() {
    if [[ $EUID -eq 0 ]]; then
        echo "/usr/local"
    else
        echo "${HOME}/.local"
    fi
}

# Get available disk space in MB for a given path
# Falls back to parent directories if path doesn't exist
get_available_disk_space() {
    local target_path="$1"
    local check_path="$target_path"

    # Find an existing directory to check (walk up the tree)
    while [[ ! -d "$check_path" ]] && [[ "$check_path" != "/" ]]; do
        check_path=$(dirname "$check_path")
    done

    # If we couldn't find any existing directory, use root
    if [[ ! -d "$check_path" ]]; then
        check_path="/"
    fi

    local available_space
    if [[ "$OS" == "darwin" ]]; then
        # macOS: df -m output has "Available" in column 4
        # Format: Filesystem 1M-blocks Used Available Capacity iused ifree %iused Mounted
        available_space=$(df -m "$check_path" 2>/dev/null | awk 'NR==2 {print $4}')
    else
        # Linux: df -m output typically has "Available" in column 4
        # Format: Filesystem 1M-blocks Used Available Use% Mounted
        available_space=$(df -m "$check_path" 2>/dev/null | awk 'NR==2 {print $4}')
    fi

    # Validate that we got a number
    if [[ "$available_space" =~ ^[0-9]+$ ]]; then
        echo "$available_space"
    else
        echo "unknown"
    fi
}

# =============================================================================
# Installation Mode Detection
# =============================================================================

# Detect if we're running from a local source directory or remotely
detect_install_mode() {
    if [[ "$INSTALL_MODE" == "source" ]]; then
        RESOLVED_INSTALL_MODE="source"
        return
    fi

    if [[ "$INSTALL_MODE" == "binary" ]]; then
        RESOLVED_INSTALL_MODE="binary"
        return
    fi

    # Auto-detect: check if we're in a FlyDB source directory
    if [[ -f "go.mod" ]] && grep -q "flydb" go.mod 2>/dev/null; then
        # We're in a source directory
        if command -v go &>/dev/null; then
            RESOLVED_INSTALL_MODE="source"
            print_info "Detected local source directory - will build from source"
        else
            print_warning "Source directory detected but Go not found - will download binaries"
            RESOLVED_INSTALL_MODE="binary"
        fi
    else
        # Not in source directory - download pre-built binaries
        RESOLVED_INSTALL_MODE="binary"
        print_info "Will download pre-built binaries from GitHub"
    fi
}

# Get the download URL for the release archive
get_download_url() {
    local version="${SPECIFIC_VERSION:-$FLYDB_VERSION}"

    # Remove 'v' prefix if present for consistency
    version="${version#v}"

    # Construct the archive name: flydb_<version>_<os>_<arch>.tar.gz
    local archive_name="flydb_${version}_${OS}_${ARCH}.tar.gz"

    echo "${DOWNLOAD_BASE_URL}/v${version}/${archive_name}"
}

# Create a temporary directory for downloads
create_temp_dir() {
    TEMP_DIR=$(mktemp -d 2>/dev/null || mktemp -d -t 'flydb-install')
    if [[ ! -d "$TEMP_DIR" ]]; then
        print_error "Failed to create temporary directory"
        exit 1
    fi
}

# Clean up temporary directory
cleanup_temp_dir() {
    if [[ -n "$TEMP_DIR" ]] && [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
        TEMP_DIR=""
    fi
}

# Download and extract pre-built binaries
download_binaries() {
    print_step "Downloading FlyDB binaries..."

    local version="${SPECIFIC_VERSION:-$FLYDB_VERSION}"
    version="${version#v}"

    local download_url
    download_url=$(get_download_url)

    create_temp_dir

    local archive_path="$TEMP_DIR/flydb.tar.gz"

    spinner_start "Downloading FlyDB v${version} for ${OS}/${ARCH}"

    local http_code
    http_code=$(curl -fsSL -w "%{http_code}" -o "$archive_path" "$download_url" 2>/dev/null) || true

    if [[ "$http_code" != "200" ]] || [[ ! -f "$archive_path" ]]; then
        spinner_error "Failed to download binaries (HTTP $http_code)"
        echo ""
        print_error "Could not download from: $download_url"
        echo ""
        print_info "Possible solutions:"
        echo "  1. Check if version v${version} exists for ${OS}/${ARCH}"
        echo "  2. Check your internet connection"
        echo "  3. Build from source: git clone https://github.com/${GITHUB_REPO}.git && cd flydb && ./install.sh --from-source"
        cleanup_temp_dir
        exit 1
    fi

    spinner_success "Downloaded FlyDB v${version}"

    spinner_start "Extracting binaries"

    if ! tar -xzf "$archive_path" -C "$TEMP_DIR" 2>/dev/null; then
        spinner_error "Failed to extract archive"
        cleanup_temp_dir
        exit 1
    fi

    # Verify extracted binaries exist
    if [[ ! -f "$TEMP_DIR/flydb" ]] || [[ ! -f "$TEMP_DIR/flydb-shell" ]]; then
        spinner_error "Archive does not contain expected binaries"
        cleanup_temp_dir
        exit 1
    fi

    spinner_success "Extracted binaries"
    echo ""
}

# Install binaries from downloaded files
install_downloaded_binaries() {
    print_step "Installing binaries..."

    INSTALL_STARTED=true

    local bin_dir="${PREFIX}/bin"
    local sudo_cmd
    sudo_cmd=$(get_sudo_cmd "$bin_dir")

    # Create bin directory
    if [[ ! -d "$bin_dir" ]]; then
        spinner_start "Creating directory $bin_dir"
        if $sudo_cmd mkdir -p "$bin_dir" 2>/dev/null; then
            spinner_success "Created $bin_dir"
            CREATED_DIRS+=("$bin_dir")
        else
            spinner_error "Failed to create $bin_dir"
            cleanup_temp_dir
            exit 1
        fi
    else
        print_substep "Directory exists: $bin_dir"
    fi

    # Install flydb
    spinner_start "Installing flydb"
    if $sudo_cmd cp "$TEMP_DIR/flydb" "$bin_dir/" && $sudo_cmd chmod +x "$bin_dir/flydb"; then
        spinner_success "Installed ${bin_dir}/flydb"
        INSTALLED_FILES+=("$bin_dir/flydb")
    else
        spinner_error "Failed to install flydb"
        cleanup_temp_dir
        rollback
        exit 1
    fi

    # Install flydb-shell
    spinner_start "Installing flydb-shell"
    if $sudo_cmd cp "$TEMP_DIR/flydb-shell" "$bin_dir/" && $sudo_cmd chmod +x "$bin_dir/flydb-shell"; then
        spinner_success "Installed ${bin_dir}/flydb-shell"
        INSTALLED_FILES+=("$bin_dir/flydb-shell")
    else
        spinner_error "Failed to install flydb-shell"
        cleanup_temp_dir
        rollback
        exit 1
    fi

    # Create fsql symlink for convenience
    spinner_start "Creating fsql symlink"
    if $sudo_cmd ln -sf "$bin_dir/flydb-shell" "$bin_dir/fsql"; then
        spinner_success "Created ${bin_dir}/fsql symlink"
        INSTALLED_FILES+=("$bin_dir/fsql")
    else
        spinner_error "Failed to create fsql symlink"
    fi

    # Clean up temp directory
    cleanup_temp_dir

    echo ""
}

# =============================================================================
# Prerequisite Checks
# =============================================================================

check_prerequisites() {
    print_step "Checking prerequisites..."

    local errors=0

    # Check for required commands based on installation mode
    local required_commands=("curl" "tar")
    if [[ "$RESOLVED_INSTALL_MODE" == "source" ]]; then
        required_commands+=("go")
    fi

    for cmd in "${required_commands[@]}"; do
        if command -v "$cmd" &>/dev/null; then
            print_substep "${GREEN}${ICON_SUCCESS}${RESET} $cmd found"
        else
            print_substep "${RED}${ICON_ERROR}${RESET} $cmd not found"
            ((errors++))
        fi
    done

    # Check Go version if building from source
    if [[ "$RESOLVED_INSTALL_MODE" == "source" ]] && command -v go &>/dev/null; then
        local go_version
        go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
        local go_major go_minor
        go_major=$(echo "$go_version" | cut -d. -f1)
        go_minor=$(echo "$go_version" | cut -d. -f2)

        if [[ "$go_major" -lt 1 ]] || ([[ "$go_major" -eq 1 ]] && [[ "$go_minor" -lt 21 ]]); then
            print_substep "${RED}${ICON_ERROR}${RESET} Go 1.21+ required (found: $go_version)"
            ((errors++))
        else
            print_substep "${GREEN}${ICON_SUCCESS}${RESET} Go $go_version"
        fi
    fi

    # Check disk space (require at least 100MB)
    local install_dir="${PREFIX:-$(get_default_prefix)}"
    local available_space
    available_space=$(get_available_disk_space "$install_dir")

    if [[ "$available_space" == "unknown" ]]; then
        print_substep "${YELLOW}${ICON_WARNING}${RESET} Could not determine available disk space"
    elif [[ "$available_space" -lt 100 ]]; then
        print_substep "${YELLOW}${ICON_WARNING}${RESET} Low disk space: ${available_space}MB available"
    else
        print_substep "${GREEN}${ICON_SUCCESS}${RESET} Disk space: ${available_space}MB available"
    fi

    # Check write permissions
    local test_dir="${install_dir}/bin"
    if [[ -d "$test_dir" ]]; then
        if [[ -w "$test_dir" ]]; then
            print_substep "${GREEN}${ICON_SUCCESS}${RESET} Write permission to $test_dir"
        else
            if [[ $EUID -eq 0 ]]; then
                print_substep "${RED}${ICON_ERROR}${RESET} No write permission to $test_dir"
                ((errors++))
            else
                print_substep "${YELLOW}${ICON_WARNING}${RESET} Will need sudo for $test_dir"
            fi
        fi
    else
        # Directory doesn't exist yet, check parent
        local parent_dir
        parent_dir=$(dirname "$test_dir")
        if [[ -d "$parent_dir" ]] && [[ -w "$parent_dir" ]]; then
            print_substep "${GREEN}${ICON_SUCCESS}${RESET} Can create $test_dir"
        elif [[ $EUID -eq 0 ]]; then
            print_substep "${GREEN}${ICON_SUCCESS}${RESET} Can create $test_dir (as root)"
        else
            print_substep "${YELLOW}${ICON_WARNING}${RESET} Will need sudo to create $test_dir"
        fi
    fi

    if [[ $errors -gt 0 ]]; then
        echo ""
        print_error "Prerequisites check failed with $errors error(s)"
        exit 1
    fi

    echo ""
}

check_existing_installation() {
    local install_dir="${PREFIX:-$(get_default_prefix)}/bin"
    local existing_version=""

    if [[ -x "$install_dir/flydb" ]]; then
        existing_version=$("$install_dir/flydb" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
        return 0
    fi

    # Also check common locations
    for dir in /usr/local/bin /usr/bin ~/.local/bin; do
        if [[ -x "$dir/flydb" ]]; then
            existing_version=$("$dir/flydb" --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo "unknown")
            echo "$dir:$existing_version"
            return 0
        fi
    done

    return 1
}

# =============================================================================
# Interactive Wizard
# =============================================================================

prompt() {
    local prompt_text="$1"
    local default="${2:-}"
    local result=""

    # Ensure any spinner is stopped before prompting
    ensure_clean_prompt

    if [[ -n "$default" ]]; then
        echo -en "${BOLD}${prompt_text}${RESET} [${CYAN}${default}${RESET}]: " >&2
    else
        echo -en "${BOLD}${prompt_text}${RESET}: " >&2
    fi

    read -r result

    if [[ -z "$result" ]]; then
        result="$default"
    fi

    echo "$result"
}

prompt_yes_no() {
    local prompt_text="$1"
    local default="${2:-y}"
    local result=""

    # Ensure any spinner is stopped before prompting
    ensure_clean_prompt

    local hint
    if [[ "$default" == "y" ]]; then
        hint="Y/n"
    else
        hint="y/N"
    fi

    echo -en "${BOLD}${prompt_text}${RESET} [${CYAN}${hint}${RESET}]: " >&2
    read -r result

    if [[ -z "$result" ]]; then
        result="$default"
    fi

    # Convert to lowercase using tr for POSIX compatibility (macOS default bash is 3.2)
    local lower_result
    lower_result=$(echo "$result" | tr '[:upper:]' '[:lower:]')
    
    case "$lower_result" in
        y|yes) return 0 ;;
        *) return 1 ;;
    esac
}

validate_path() {
    local path="$1"

    # Expand ~ to home directory
    path="${path/#\~/$HOME}"

    # Check if path is absolute or can be made absolute
    if [[ ! "$path" = /* ]]; then
        path="$(pwd)/$path"
    fi

    echo "$path"
}

validate_port() {
    local port="$1"
    if [[ "$port" =~ ^[0-9]+$ ]] && [[ "$port" -ge 1 ]] && [[ "$port" -le 65535 ]]; then
        return 0
    fi
    return 1
}

run_interactive_wizard() {
    echo "Welcome to FlyDB! This wizard will guide you through the installation."
    echo ""
    echo -e "Press ${CYAN}Enter${RESET} to accept default values shown in [brackets]."
    echo -e "Press ${CYAN}Ctrl+C${RESET} to cancel at any time."
    echo ""

    # Check for existing installation
    local existing
    if existing=$(check_existing_installation 2>/dev/null); then
        local existing_path existing_ver
        existing_path=$(echo "$existing" | cut -d: -f1)
        existing_ver=$(echo "$existing" | cut -d: -f2)

        print_warning "Existing FlyDB installation detected"
        print_kv "Location" "$existing_path"
        print_kv "Version" "$existing_ver"
        echo ""

        if ! prompt_yes_no "Would you like to upgrade/reinstall?"; then
            echo ""
            print_info "Installation cancelled"
            exit 0
        fi
        echo ""
    fi

    # Step 1: Installation Directory
    echo -e "${CYAN}${BOLD}Step 1: Installation Directory${RESET}"
    separator 60
    echo ""
    echo "  Where should FlyDB be installed?"
    echo ""
    echo -e "  ${GREEN}1${RESET}) /usr/local/bin  (system-wide, requires sudo)"
    echo -e "  ${GREEN}2${RESET}) ~/.local/bin    (user-only, no sudo required)"
    echo -e "  ${GREEN}3${RESET}) Custom path"
    echo ""

    local choice
    choice=$(prompt "Select option [1-3]" "1")
    # Trim any whitespace
    choice="${choice//[[:space:]]/}"

    case "$choice" in
        1) PREFIX="/usr/local" ;;
        2) PREFIX="$HOME/.local" ;;
        3)
            local custom_path
            custom_path=$(prompt "Enter installation path" "/opt/flydb")
            PREFIX=$(validate_path "$custom_path")
            ;;
        *)
            print_warning "Invalid selection, using default"
            PREFIX="/usr/local"
            ;;
    esac
    echo ""

    # Step 2: System Service
    echo -e "${CYAN}${BOLD}Step 2: System Service${RESET}"
    separator 60
    echo ""

    if [[ "$INIT_SYSTEM" != "none" ]]; then
        echo "  FlyDB can be installed as a system service ($INIT_SYSTEM)"
        echo "  This allows FlyDB to start automatically on boot."
        echo ""

        if prompt_yes_no "Install as system service?"; then
            INSTALL_SERVICE=true
        else
            INSTALL_SERVICE=false
        fi
    else
        print_warning "No supported init system detected, skipping service installation"
        INSTALL_SERVICE=false
    fi
    echo ""

    # Step 3: Configuration
    echo -e "${CYAN}${BOLD}Step 3: Configuration${RESET}"
    separator 60
    echo ""
    echo "  FlyDB can create a default configuration file."
    echo ""

    if prompt_yes_no "Create default configuration file?"; then
        CREATE_CONFIG=true
    else
        CREATE_CONFIG=false
    fi
    echo ""

    # Step 4: Initialize Database
    echo -e "${CYAN}${BOLD}Step 4: Database Initialization${RESET}"
    separator 60
    echo ""
    echo "  FlyDB can initialize a new database during installation."
    echo ""

    if prompt_yes_no "Initialize a new database?" "n"; then
        INIT_DATABASE=true
    else
        INIT_DATABASE=false
    fi
    echo ""

    # Summary
    print_installation_summary

    # Confirmation
    echo ""
    if ! prompt_yes_no "Proceed with installation?"; then
        echo ""
        print_info "Installation cancelled"
        exit 0
    fi
    echo ""
}

print_installation_summary() {
    echo ""
    echo -e "${CYAN}${BOLD}Installation Summary${RESET}"
    double_separator 60
    echo ""

    local version="${SPECIFIC_VERSION:-$FLYDB_VERSION}"
    version="${version#v}"
    print_kv "FlyDB Version" "$version"
    print_kv "Operating System" "$OS ($DISTRO)"
    print_kv "Architecture" "$ARCH"
    print_kv "Install Directory" "${PREFIX}/bin"

    # Show installation mode
    if [[ "$RESOLVED_INSTALL_MODE" == "source" ]]; then
        print_kv "Install Method" "${CYAN}Build from source${RESET}"
    else
        print_kv "Install Method" "${CYAN}Download binaries${RESET}"
    fi

    if [[ "$INSTALL_SERVICE" == true ]]; then
        print_kv "System Service" "${GREEN}Yes${RESET} ($INIT_SYSTEM)"
    else
        print_kv "System Service" "${DIM}No${RESET}"
    fi

    if [[ "$CREATE_CONFIG" == true ]]; then
        print_kv "Create Config" "${GREEN}Yes${RESET}"
    else
        print_kv "Create Config" "${DIM}No${RESET}"
    fi

    if [[ "$INIT_DATABASE" == true ]]; then
        print_kv "Init Database" "${GREEN}Yes${RESET}"
    else
        print_kv "Init Database" "${DIM}No${RESET}"
    fi

    echo ""
}

# =============================================================================
# Installation Functions
# =============================================================================

# Track if we've already obtained sudo credentials
SUDO_OBTAINED=false
SUDO_KEEPALIVE_PID=""

get_sudo_cmd() {
    local target_dir="$1"

    if [[ -w "$target_dir" ]] || [[ -w "$(dirname "$target_dir")" ]]; then
        echo ""
    elif [[ $EUID -eq 0 ]]; then
        echo ""
    else
        echo "sudo"
    fi
}

# Check if sudo will be needed for the installation
needs_sudo() {
    local bin_dir="${PREFIX}/bin"
    local config_dir

    if [[ $EUID -eq 0 ]]; then
        return 1  # Running as root, no sudo needed
    fi

    # Check if we can write to bin directory or its parent
    if [[ -d "$bin_dir" ]]; then
        [[ ! -w "$bin_dir" ]] && return 0
    else
        [[ ! -w "$(dirname "$bin_dir")" ]] && return 0
    fi

    # Check config directory if we're creating config
    if [[ "$CREATE_CONFIG" == true ]]; then
        if [[ "$PREFIX" == "/usr/local" ]]; then
            config_dir="/etc/flydb"
            [[ ! -w "/etc" ]] && return 0
        fi
    fi

    # Check service installation
    if [[ "$INSTALL_SERVICE" == true ]]; then
        if [[ "$INIT_SYSTEM" == "systemd" ]]; then
            [[ ! -w "/etc/systemd/system" ]] && return 0
        fi
    fi

    return 1  # No sudo needed
}

# Stop the sudo keepalive background process
stop_sudo_keepalive() {
    if [[ -n "$SUDO_KEEPALIVE_PID" ]] && kill -0 "$SUDO_KEEPALIVE_PID" 2>/dev/null; then
        kill "$SUDO_KEEPALIVE_PID" 2>/dev/null || true
        wait "$SUDO_KEEPALIVE_PID" 2>/dev/null || true
        SUDO_KEEPALIVE_PID=""
    fi
}

# Obtain sudo credentials upfront if needed
obtain_sudo_if_needed() {
    if [[ "$SUDO_OBTAINED" == true ]]; then
        return 0
    fi

    if needs_sudo; then
        echo ""
        print_info "This installation requires elevated privileges (sudo)"
        echo ""

        # Prompt for sudo password before any spinners start
        if sudo -v; then
            SUDO_OBTAINED=true

            # Start a background process to keep sudo credentials alive
            # Use a simple approach that doesn't interfere with the main script
            local parent_pid=$$
            (
                # Disable errexit in subshell to prevent premature exit
                set +e
                while true; do
                    sleep 50
                    # Check if parent is still running
                    if ! kill -0 "$parent_pid" 2>/dev/null; then
                        exit 0
                    fi
                    # Refresh sudo credentials silently
                    sudo -n true 2>/dev/null || true
                done
            ) &
            SUDO_KEEPALIVE_PID=$!
            # Disown the background process so it doesn't affect script exit
            disown "$SUDO_KEEPALIVE_PID" 2>/dev/null || true
        else
            print_error "Failed to obtain sudo privileges"
            return 1
        fi
    fi

    return 0
}

build_binaries() {
    print_step "Building FlyDB from source..."

    # Verify we're in the right directory
    if [[ ! -f "go.mod" ]] || ! grep -q "flydb" go.mod 2>/dev/null; then
        print_error "Not in FlyDB source directory. Please run from the repository root."
        exit 1
    fi

    spinner_start "Building flydb daemon"
    if go build -o flydb ./cmd/flydb 2>/dev/null; then
        spinner_success "Built flydb daemon"
        INSTALLED_FILES+=("./flydb")
    else
        spinner_error "Failed to build flydb daemon"
        exit 1
    fi

    spinner_start "Building flydb-shell client"
    if go build -o flydb-shell ./cmd/flydb-shell 2>/dev/null; then
        spinner_success "Built flydb-shell client"
        INSTALLED_FILES+=("./flydb-shell")
    else
        spinner_error "Failed to build flydb-shell client"
        exit 1
    fi

    echo ""
}

install_binaries() {
    print_step "Installing binaries..."

    INSTALL_STARTED=true

    local bin_dir="${PREFIX}/bin"
    local sudo_cmd
    sudo_cmd=$(get_sudo_cmd "$bin_dir")

    # Create bin directory
    if [[ ! -d "$bin_dir" ]]; then
        spinner_start "Creating directory $bin_dir"
        if $sudo_cmd mkdir -p "$bin_dir" 2>/dev/null; then
            spinner_success "Created $bin_dir"
            CREATED_DIRS+=("$bin_dir")
        else
            spinner_error "Failed to create $bin_dir"
            exit 1
        fi
    else
        print_substep "Directory exists: $bin_dir"
    fi

    # Install flydb
    spinner_start "Installing flydb"
    if $sudo_cmd cp flydb "$bin_dir/" && $sudo_cmd chmod +x "$bin_dir/flydb"; then
        spinner_success "Installed ${bin_dir}/flydb"
        INSTALLED_FILES+=("$bin_dir/flydb")
    else
        spinner_error "Failed to install flydb"
        rollback
        exit 1
    fi

    # Install flydb-shell
    spinner_start "Installing flydb-shell"
    if $sudo_cmd cp flydb-shell "$bin_dir/" && $sudo_cmd chmod +x "$bin_dir/flydb-shell"; then
        spinner_success "Installed ${bin_dir}/flydb-shell"
        INSTALLED_FILES+=("$bin_dir/flydb-shell")
    else
        spinner_error "Failed to install flydb-shell"
        rollback
        exit 1
    fi

    # Create fsql symlink for convenience
    spinner_start "Creating fsql symlink"
    if $sudo_cmd ln -sf "$bin_dir/flydb-shell" "$bin_dir/fsql"; then
        spinner_success "Created ${bin_dir}/fsql symlink"
        INSTALLED_FILES+=("$bin_dir/fsql")
    else
        spinner_error "Failed to create fsql symlink"
    fi

    echo ""
}

create_config_file() {
    if [[ "$CREATE_CONFIG" != true ]]; then
        return
    fi

    print_step "Creating configuration file..."

    local config_dir
    local data_dir
    local sudo_cmd

    if [[ $EUID -eq 0 ]] || [[ "$PREFIX" == "/usr/local" ]]; then
        config_dir="/etc/flydb"
        data_dir="/var/lib/flydb"
        sudo_cmd=$(get_sudo_cmd "$config_dir")
    else
        config_dir="$HOME/.config/flydb"
        data_dir="$HOME/.local/share/flydb"
        sudo_cmd=""
    fi

    # Create data directory if it doesn't exist
    if [[ ! -d "$data_dir" ]]; then
        spinner_start "Creating data directory"
        if $sudo_cmd mkdir -p "$data_dir" 2>/dev/null; then
            spinner_success "Created $data_dir"
            CREATED_DIRS+=("$data_dir")
        else
            spinner_error "Failed to create data directory"
            return 1
        fi
    fi

    if [[ ! -d "$config_dir" ]]; then
        spinner_start "Creating config directory"
        if $sudo_cmd mkdir -p "$config_dir" 2>/dev/null; then
            spinner_success "Created $config_dir"
            CREATED_DIRS+=("$config_dir")
        else
            spinner_error "Failed to create config directory"
            return 1
        fi
    fi

    local config_file="$config_dir/flydb.conf"

    if [[ -f "$config_file" ]]; then
        print_warning "Configuration file already exists: $config_file"
        print_substep "Skipping config creation to preserve existing settings"
        return 0
    fi

    spinner_start "Writing configuration file"

    local config_content="# FlyDB Configuration File
# Generated by install.sh on $(date)
#
# Configuration Precedence (highest to lowest):
#   1. Command-line flags
#   2. Environment variables (FLYDB_*)
#   3. This configuration file
#   4. Default values
#
# Environment Variables:
#   FLYDB_PORT          - Server port for text protocol
#   FLYDB_BINARY_PORT   - Server port for binary protocol
#   FLYDB_REPL_PORT     - Replication port
#   FLYDB_ROLE          - Server role (standalone, master, slave, cluster)
#   FLYDB_MASTER_ADDR   - Master address for slave mode
#   FLYDB_DATA_DIR      - Data directory for database storage
#   FLYDB_LOG_LEVEL     - Log level (debug, info, warn, error)
#   FLYDB_LOG_JSON      - Enable JSON logging (true/false)
#   FLYDB_ADMIN_PASSWORD - Initial admin password (first-time setup only)
#   FLYDB_ENCRYPTION_PASSPHRASE - Encryption passphrase (required if encryption enabled)
#   FLYDB_CONFIG_FILE   - Path to this configuration file
#
# Cluster Environment Variables:
#   FLYDB_CLUSTER_PORT        - Port for cluster communication (default: 9998)
#   FLYDB_CLUSTER_PEERS       - Comma-separated list of peer addresses
#   FLYDB_REPLICATION_MODE    - Replication mode: async, semi_sync, sync
#   FLYDB_HEARTBEAT_INTERVAL_MS - Heartbeat interval in milliseconds
#   FLYDB_HEARTBEAT_TIMEOUT_MS  - Heartbeat timeout in milliseconds
#   FLYDB_ELECTION_TIMEOUT_MS   - Election timeout in milliseconds
#   FLYDB_MIN_QUORUM          - Minimum quorum size for cluster decisions

# Server role: standalone, master, slave, or cluster
# - standalone: Single server mode (no replication)
# - master: Leader node that accepts writes and replicates to slaves
# - slave: Follower node that receives replication from master
# - cluster: Automatic failover cluster with leader election
role = \"standalone\"

# Network ports
# Text protocol port (for nc/telnet connections)
port = 8888
# Binary protocol port (for fsql connections)
binary_port = 8889
# Replication port (master mode only)
replication_port = 9999

# Master address for slave mode (format: host:port)
# Uncomment and set when running in slave mode
# master_addr = \"localhost:9999\"

# Storage
# Data directory for multi-database storage
# User installations: ~/.local/share/flydb
# System installations: /var/lib/flydb
data_dir = \"$data_dir\"

# Logging
# Available levels: debug, info, warn, error
log_level = \"info\"
# Enable JSON-formatted log output (useful for log aggregation)
log_json = false

# Authentication
# Note: Admin password is set on first run via:
#   - FLYDB_ADMIN_PASSWORD environment variable, or
#   - Interactive wizard (if no env var set)

# =============================================================================
# Cluster Configuration (for role = \"cluster\")
# =============================================================================

# Cluster communication port
# cluster_port = 9998

# Comma-separated list of peer node addresses (host:port)
# Example: cluster_peers = [\"node2:9998\", \"node3:9998\"]
# cluster_peers = []

# Replication mode: async, semi_sync, or sync
# - async: Best performance, eventual consistency
# - semi_sync: At least one replica acknowledges before commit
# - sync: All replicas must acknowledge (strongest consistency)
replication_mode = \"async\"

# Heartbeat interval in milliseconds (how often to send heartbeats)
heartbeat_interval_ms = 1000

# Heartbeat timeout in milliseconds (when to consider a node dead)
heartbeat_timeout_ms = 5000

# Election timeout in milliseconds (when to start a new election)
election_timeout_ms = 10000

# Minimum quorum size for cluster decisions
# Set to 0 for automatic calculation (majority of nodes)
min_quorum = 0

# Enable pre-vote protocol to prevent disruptions from partitioned nodes
enable_pre_vote = true

# Sync timeout in milliseconds (for sync replication mode)
sync_timeout_ms = 5000

# Maximum replication lag in bytes before a replica is considered unhealthy
max_replication_lag = 10485760
"

    if echo "$config_content" | $sudo_cmd tee "$config_file" >/dev/null 2>&1; then
        spinner_success "Created $config_file"
        INSTALLED_FILES+=("$config_file")
    else
        spinner_error "Failed to create configuration file"
        return 1
    fi

    echo ""
}

install_systemd_service() {
    if [[ "$INSTALL_SERVICE" != true ]] || [[ "$INIT_SYSTEM" != "systemd" ]]; then
        return
    fi

    print_step "Installing systemd service..."

    local service_file="/etc/systemd/system/flydb.service"
    local sudo_cmd
    sudo_cmd=$(get_sudo_cmd "/etc/systemd/system")

    if [[ -f "$service_file" ]]; then
        print_warning "Service file already exists: $service_file"
        if ! prompt_yes_no "Overwrite existing service file?" "n"; then
            print_substep "Skipping service installation"
            return 0
        fi
    fi

    local service_content="[Unit]
Description=FlyDB Database Server
Documentation=https://flydb.dev/docs
After=network.target

[Service]
Type=simple
User=flydb
Group=flydb
ExecStart=${PREFIX}/bin/flydb
Restart=on-failure
RestartSec=5
LimitNOFILE=65535

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/flydb

[Install]
WantedBy=multi-user.target
"

    spinner_start "Creating systemd service"
    if echo "$service_content" | $sudo_cmd tee "$service_file" >/dev/null 2>&1; then
        spinner_success "Created $service_file"
        INSTALLED_FILES+=("$service_file")
    else
        spinner_error "Failed to create service file"
        return 1
    fi

    # Create flydb user if it doesn't exist
    if ! id flydb &>/dev/null; then
        spinner_start "Creating flydb system user"
        if $sudo_cmd useradd --system --no-create-home --shell /usr/sbin/nologin flydb 2>/dev/null; then
            spinner_success "Created flydb user"
        else
            spinner_error "Failed to create flydb user"
        fi
    fi

    # Create data directory
    local data_dir="/var/lib/flydb"
    if [[ ! -d "$data_dir" ]]; then
        spinner_start "Creating data directory"
        if $sudo_cmd mkdir -p "$data_dir" && $sudo_cmd chown flydb:flydb "$data_dir" 2>/dev/null; then
            spinner_success "Created $data_dir"
            CREATED_DIRS+=("$data_dir")
        else
            spinner_error "Failed to create data directory"
        fi
    fi

    # Reload systemd
    spinner_start "Reloading systemd"
    if $sudo_cmd systemctl daemon-reload 2>/dev/null; then
        spinner_success "Reloaded systemd"
    else
        spinner_error "Failed to reload systemd"
    fi

    echo ""
}

install_launchd_service() {
    if [[ "$INSTALL_SERVICE" != true ]] || [[ "$INIT_SYSTEM" != "launchd" ]]; then
        return
    fi

    print_step "Installing launchd service..."

    local plist_dir
    local plist_file
    local sudo_cmd

    if [[ $EUID -eq 0 ]]; then
        plist_dir="/Library/LaunchDaemons"
        plist_file="$plist_dir/io.flydb.flydb.plist"
        sudo_cmd=""
    else
        plist_dir="$HOME/Library/LaunchAgents"
        plist_file="$plist_dir/io.flydb.flydb.plist"
        sudo_cmd=""
        mkdir -p "$plist_dir"
    fi

    if [[ -f "$plist_file" ]]; then
        print_warning "Plist file already exists: $plist_file"
        if ! prompt_yes_no "Overwrite existing plist file?" "n"; then
            print_substep "Skipping service installation"
            return 0
        fi
    fi

    local plist_content="<?xml version=\"1.0\" encoding=\"UTF-8\"?>
<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">
<plist version=\"1.0\">
<dict>
    <key>Label</key>
    <string>io.flydb.flydb</string>
    <key>ProgramArguments</key>
    <array>
        <string>${PREFIX}/bin/flydb</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/var/log/flydb/error.log</string>
    <key>StandardOutPath</key>
    <string>/var/log/flydb/output.log</string>
</dict>
</plist>
"

    spinner_start "Creating launchd plist"
    if echo "$plist_content" | $sudo_cmd tee "$plist_file" >/dev/null 2>&1; then
        spinner_success "Created $plist_file"
        INSTALLED_FILES+=("$plist_file")
    else
        spinner_error "Failed to create plist file"
        return 1
    fi

    # Create log directory
    local log_dir="/var/log/flydb"
    if [[ ! -d "$log_dir" ]]; then
        spinner_start "Creating log directory"
        local log_sudo
        log_sudo=$(get_sudo_cmd "$log_dir")
        if $log_sudo mkdir -p "$log_dir" 2>/dev/null; then
            spinner_success "Created $log_dir"
            CREATED_DIRS+=("$log_dir")
        else
            print_warning "Could not create log directory: $log_dir"
        fi
    fi

    echo ""
}

verify_installation() {
    print_step "Verifying installation..."

    local bin_dir="${PREFIX}/bin"
    local errors=0

    # Check flydb binary
    if [[ -x "$bin_dir/flydb" ]]; then
        local version
        version=$("$bin_dir/flydb" --version 2>/dev/null | head -1 || echo "unknown")
        print_substep "${GREEN}${ICON_SUCCESS}${RESET} flydb: $version"
    else
        print_substep "${RED}${ICON_ERROR}${RESET} flydb: not found or not executable"
        ((errors++))
    fi

    # Check flydb-shell binary
    if [[ -x "$bin_dir/flydb-shell" ]]; then
        local version
        version=$("$bin_dir/flydb-shell" --version 2>/dev/null | head -1 || echo "unknown")
        print_substep "${GREEN}${ICON_SUCCESS}${RESET} flydb-shell: $version"
    else
        print_substep "${RED}${ICON_ERROR}${RESET} flydb-shell: not found or not executable"
        ((errors++))
    fi

    # Check fsql symlink
    if [[ -x "$bin_dir/fsql" ]]; then
        print_substep "${GREEN}${ICON_SUCCESS}${RESET} fsql: symlink OK"
    else
        print_substep "${YELLOW}${ICON_WARNING}${RESET} fsql: symlink not found"
    fi

    echo ""

    if [[ $errors -gt 0 ]]; then
        print_error "Installation verification failed"
        return 1
    fi

    return 0
}

# =============================================================================
# Uninstallation
# =============================================================================

run_uninstall() {
    print_step "Uninstalling FlyDB..."
    echo ""

    local found=false

    # Find installation locations
    local locations=("/usr/local/bin" "/usr/bin" "$HOME/.local/bin")
    if [[ -n "$PREFIX" ]]; then
        locations=("${PREFIX}/bin" "${locations[@]}")
    fi

    for dir in "${locations[@]}"; do
        if [[ -x "$dir/flydb" ]] || [[ -x "$dir/flydb-shell" ]]; then
            print_info "Found FlyDB installation in $dir"
            found=true

            if [[ "$AUTO_CONFIRM" != true ]]; then
                if ! prompt_yes_no "Remove files from $dir?"; then
                    continue
                fi
            fi

            local sudo_cmd
            sudo_cmd=$(get_sudo_cmd "$dir")

            if [[ -x "$dir/flydb" ]]; then
                spinner_start "Removing flydb"
                if $sudo_cmd rm -f "$dir/flydb" 2>/dev/null; then
                    spinner_success "Removed $dir/flydb"
                else
                    spinner_error "Failed to remove $dir/flydb"
                fi
            fi

            if [[ -x "$dir/flydb-shell" ]]; then
                spinner_start "Removing flydb-shell"
                if $sudo_cmd rm -f "$dir/flydb-shell" 2>/dev/null; then
                    spinner_success "Removed $dir/flydb-shell"
                else
                    spinner_error "Failed to remove $dir/flydb-shell"
                fi
            fi

            if [[ -L "$dir/fsql" ]] || [[ -x "$dir/fsql" ]]; then
                spinner_start "Removing fsql symlink"
                if $sudo_cmd rm -f "$dir/fsql" 2>/dev/null; then
                    spinner_success "Removed $dir/fsql"
                else
                    spinner_error "Failed to remove $dir/fsql"
                fi
            fi
        fi
    done

    # Remove systemd service
    if [[ -f "/etc/systemd/system/flydb.service" ]]; then
        print_info "Found systemd service"

        if [[ "$AUTO_CONFIRM" != true ]]; then
            if prompt_yes_no "Remove systemd service?"; then
                local sudo_cmd
                sudo_cmd=$(get_sudo_cmd "/etc/systemd/system")
                $sudo_cmd systemctl stop flydb 2>/dev/null || true
                $sudo_cmd systemctl disable flydb 2>/dev/null || true
                $sudo_cmd rm -f /etc/systemd/system/flydb.service
                $sudo_cmd systemctl daemon-reload 2>/dev/null || true
                print_success "Removed systemd service"
            fi
        fi
    fi

    # Remove launchd service
    local plist_files=(
        "/Library/LaunchDaemons/io.flydb.flydb.plist"
        "$HOME/Library/LaunchAgents/io.flydb.flydb.plist"
    )

    for plist in "${plist_files[@]}"; do
        if [[ -f "$plist" ]]; then
            print_info "Found launchd service: $plist"

            if [[ "$AUTO_CONFIRM" != true ]]; then
                if prompt_yes_no "Remove launchd service?"; then
                    launchctl unload "$plist" 2>/dev/null || true
                    rm -f "$plist"
                    print_success "Removed launchd service"
                fi
            fi
        fi
    done

    # Remove config files
    local config_dirs=("/etc/flydb" "$HOME/.config/flydb")
    for config_dir in "${config_dirs[@]}"; do
        if [[ -d "$config_dir" ]]; then
            print_info "Found configuration in $config_dir"

            if [[ "$AUTO_CONFIRM" != true ]]; then
                if prompt_yes_no "Remove configuration files?"; then
                    local sudo_cmd
                    sudo_cmd=$(get_sudo_cmd "$config_dir")
                    $sudo_cmd rm -rf "$config_dir"
                    print_success "Removed $config_dir"
                fi
            fi
        fi
    done

    if [[ "$found" == false ]]; then
        print_warning "No FlyDB installation found"
    else
        echo ""
        print_success "FlyDB uninstallation complete"
    fi

    # Note about data
    if [[ -d "/var/lib/flydb" ]]; then
        echo ""
        print_warning "Data directory /var/lib/flydb was preserved"
        print_dim "  Remove manually if no longer needed: sudo rm -rf /var/lib/flydb"
    fi
}

# =============================================================================
# Rollback
# =============================================================================

rollback() {
    if [[ "$INSTALL_STARTED" != true ]]; then
        return
    fi

    echo ""
    print_warning "Rolling back installation..."

    # Remove installed files
    for file in "${INSTALLED_FILES[@]}"; do
        if [[ -f "$file" ]]; then
            local sudo_cmd
            sudo_cmd=$(get_sudo_cmd "$(dirname "$file")")
            $sudo_cmd rm -f "$file" 2>/dev/null && print_substep "Removed $file"
        fi
    done

    # Remove created directories (only if empty)
    for dir in "${CREATED_DIRS[@]}"; do
        if [[ -d "$dir" ]] && [[ -z "$(ls -A "$dir")" ]]; then
            local sudo_cmd
            sudo_cmd=$(get_sudo_cmd "$dir")
            $sudo_cmd rmdir "$dir" 2>/dev/null && print_substep "Removed $dir"
        fi
    done

    print_info "Rollback complete"
}

# =============================================================================
# Post-Installation
# =============================================================================

print_post_install() {
    echo ""
    echo -e "${GREEN}${BOLD}╔════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${GREEN}${BOLD}║              Installation Complete!                        ║${RESET}"
    echo -e "${GREEN}${BOLD}╚════════════════════════════════════════════════════════════╝${RESET}"
    echo ""

    local bin_dir="${PREFIX}/bin"
    local in_path=false

    # Check if bin_dir is in PATH
    if echo "$PATH" | tr ':' '\n' | grep -q "^${bin_dir}$"; then
        in_path=true
    fi

    echo -e "${BOLD}Next Steps:${RESET}"
    echo ""

    local step_num=1

    # Step: Add to PATH (only if not already in PATH)
    if [[ "$in_path" != true ]]; then
        echo -e "  ${YELLOW}${step_num}. Add FlyDB to your PATH:${RESET}"
        echo ""
        echo -e "     ${DIM}# Add to ~/.bashrc or ~/.zshrc:${RESET}"
        echo -e "     ${CYAN}export PATH=\"${bin_dir}:\$PATH\"${RESET}"
        echo ""
        ((step_num++))
    fi

    # Step: Start FlyDB
    echo -e "  ${YELLOW}${step_num}. Start FlyDB:${RESET}"
    echo ""
    echo -e "     ${DIM}# Interactive wizard (first-time setup):${RESET}"
    if [[ "$in_path" == true ]]; then
        echo -e "     ${CYAN}flydb${RESET}"
    else
        echo -e "     ${CYAN}${bin_dir}/flydb${RESET}"
    fi
    echo ""
    echo -e "     ${DIM}# Or with command-line options:${RESET}"
    if [[ "$in_path" == true ]]; then
        echo -e "     ${CYAN}flydb -port 8888 -role standalone${RESET}"
    else
        echo -e "     ${CYAN}${bin_dir}/flydb -port 8888 -role standalone${RESET}"
    fi
    ((step_num++))

    # Step: Manage service (only if service was installed)
    if [[ "$INSTALL_SERVICE" == true ]]; then
        echo ""
        echo -e "  ${YELLOW}${step_num}. Manage the service:${RESET}"
        echo ""

        if [[ "$INIT_SYSTEM" == "systemd" ]]; then
            echo -e "     ${CYAN}sudo systemctl start flydb${RESET}    # Start the service"
            echo -e "     ${CYAN}sudo systemctl enable flydb${RESET}   # Enable at boot"
            echo -e "     ${CYAN}sudo systemctl status flydb${RESET}   # Check status"
        elif [[ "$INIT_SYSTEM" == "launchd" ]]; then
            if [[ $EUID -eq 0 ]]; then
                echo -e "     ${CYAN}sudo launchctl load /Library/LaunchDaemons/io.flydb.flydb.plist${RESET}"
            else
                echo -e "     ${CYAN}launchctl load ~/Library/LaunchAgents/io.flydb.flydb.plist${RESET}"
            fi
        fi
        ((step_num++))
    fi

    # Step: Connect with CLI
    echo ""
    echo -e "  ${YELLOW}${step_num}. Connect with the CLI client:${RESET}"
    echo ""
    if [[ "$in_path" == true ]]; then
        echo -e "     ${CYAN}fsql${RESET}"
    else
        echo -e "     ${CYAN}${bin_dir}/fsql${RESET}"
    fi
    ((step_num++))

    # Step: Cluster mode (optional)
    echo ""
    echo -e "  ${YELLOW}${step_num}. Set up a cluster (optional):${RESET}"
    echo ""
    echo -e "     ${DIM}# Start a 3-node cluster with automatic failover:${RESET}"
    if [[ "$in_path" == true ]]; then
        echo -e "     ${CYAN}flydb -role cluster -cluster-peers node2:9998,node3:9998${RESET}"
    else
        echo -e "     ${CYAN}${bin_dir}/flydb -role cluster -cluster-peers node2:9998,node3:9998${RESET}"
    fi
    echo ""
    echo -e "     ${DIM}# Or use environment variables:${RESET}"
    echo -e "     ${CYAN}FLYDB_ROLE=cluster FLYDB_CLUSTER_PEERS=node2:9998,node3:9998 flydb${RESET}"

    echo ""
    separator 60
    echo ""
    echo -e "  ${DIM}Documentation:${RESET}  https://flydb.dev/docs"
    echo -e "  ${DIM}GitHub:${RESET}         https://github.com/${GITHUB_REPO}"
    echo -e "  ${DIM}Issues:${RESET}         https://github.com/${GITHUB_REPO}/issues"
    echo ""
}

# =============================================================================
# Argument Parsing
# =============================================================================

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --prefix)
                if [[ -n "${2:-}" ]]; then
                    PREFIX=$(validate_path "$2")
                    shift 2
                else
                    print_error "--prefix requires a path argument"
                    exit 1
                fi
                ;;
            --prefix=*)
                PREFIX=$(validate_path "${1#*=}")
                shift
                ;;
            --version)
                if [[ -n "${2:-}" ]]; then
                    SPECIFIC_VERSION="$2"
                    shift 2
                else
                    print_error "--version requires a version argument"
                    exit 1
                fi
                ;;
            --version=*)
                SPECIFIC_VERSION="${1#*=}"
                shift
                ;;
            --no-service)
                INSTALL_SERVICE=false
                shift
                ;;
            --no-config)
                CREATE_CONFIG=false
                shift
                ;;
            --init-db)
                INIT_DATABASE=true
                shift
                ;;
            --from-source)
                INSTALL_MODE="source"
                shift
                ;;
            --from-binary)
                INSTALL_MODE="binary"
                shift
                ;;
            --yes|-y)
                AUTO_CONFIRM=true
                INTERACTIVE=false
                shift
                ;;
            --uninstall)
                UNINSTALL=true
                shift
                ;;
            --help|-h)
                print_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo ""
                echo "Use --help to see available options"
                exit 1
                ;;
        esac
    done

    # If any argument was provided, assume non-interactive unless it's just --uninstall
    if [[ -n "$PREFIX" ]] || [[ -n "$SPECIFIC_VERSION" ]] || \
       [[ "$INSTALL_SERVICE" == false ]] || [[ "$CREATE_CONFIG" == false ]] || \
       [[ "$INIT_DATABASE" == true ]] || [[ "$AUTO_CONFIRM" == true ]] || \
       [[ "$INSTALL_MODE" != "auto" ]]; then
        INTERACTIVE=false
    fi
}

# =============================================================================
# Cleanup Handler
# =============================================================================

cleanup() {
    local exit_code=$?

    # Stop any running spinner
    spinner_stop

    # Stop sudo keepalive background process
    stop_sudo_keepalive

    # Clean up temporary directory
    cleanup_temp_dir

    # Only rollback if we were in the middle of installation and there was an error
    if [[ "$INSTALL_STARTED" == true ]] && [[ $exit_code -ne 0 ]]; then
        rollback
    fi
}

# =============================================================================
# Main
# =============================================================================

main() {
    # Set up signal handlers
    trap cleanup EXIT
    trap 'echo ""; print_warning "Installation cancelled"; exit 130' INT TERM

    # Print banner
    print_banner

    # Parse command-line arguments
    parse_args "$@"

    # Handle uninstall
    if [[ "$UNINSTALL" == true ]]; then
        run_uninstall
        exit 0
    fi

    # Detect system
    detect_os
    detect_arch
    detect_init_system

    # Detect installation mode (source vs binary)
    detect_install_mode

    # Set default prefix if not specified
    if [[ -z "$PREFIX" ]]; then
        PREFIX=$(get_default_prefix)
    fi

    # Run interactive wizard or show summary
    if [[ "$INTERACTIVE" == true ]]; then
        run_interactive_wizard
    else
        print_installation_summary

        if [[ "$AUTO_CONFIRM" != true ]]; then
            echo ""
            if ! prompt_yes_no "Proceed with installation?"; then
                print_info "Installation cancelled"
                exit 0
            fi
            echo ""
        fi
    fi

    # Check prerequisites
    check_prerequisites

    # Obtain sudo credentials upfront if needed (before any spinners)
    if ! obtain_sudo_if_needed; then
        exit 1
    fi

    # Build or download binaries based on installation mode
    if [[ "$RESOLVED_INSTALL_MODE" == "source" ]]; then
        # Build from source
        build_binaries
        # Install binaries from local build
        install_binaries
    else
        # Download pre-built binaries
        download_binaries
        # Install downloaded binaries
        install_downloaded_binaries
    fi

    # Create config file
    create_config_file

    # Install service
    if [[ "$INIT_SYSTEM" == "systemd" ]]; then
        install_systemd_service
    elif [[ "$INIT_SYSTEM" == "launchd" ]]; then
        install_launchd_service
    fi

    # Verify installation
    verify_installation

    # Print post-install instructions
    print_post_install
}

main "$@"
