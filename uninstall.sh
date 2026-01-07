#!/bin/bash
#
# FlyDB Uninstallation Script
# Copyright (c) 2026 Firefly Software Solutions Inc.
# Licensed under the Apache License, Version 2.0
#
# Usage:
#   Interactive:     ./uninstall.sh
#   Non-interactive: ./uninstall.sh --yes
#   Dry run:         ./uninstall.sh --dry-run
#

set -euo pipefail

readonly SCRIPT_VERSION="01.26.1"

PREFIX=""
AUTO_CONFIRM=false
DRY_RUN=false
REMOVE_CONFIG=true
REMOVE_DATA=false
VERBOSE=false
OS=""
INIT_SYSTEM=""

declare -a FOUND_BINARIES=()
declare -a FOUND_SERVICES=()
declare -a FOUND_CONFIGS=()
declare -a FOUND_DATA=()

SUDO_OBTAINED=false
SUDO_KEEPALIVE_PID=""

# Colors
if [[ -t 1 ]] && [[ "${TERM:-}" != "dumb" ]]; then
    RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
    BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'
    DIM='\033[2m'; RESET='\033[0m'
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; CYAN=''; BOLD=''; DIM=''; RESET=''
fi

ICON_SUCCESS="✓"; ICON_ERROR="✗"; ICON_WARNING="⚠"; ICON_INFO="ℹ"; ICON_ARROW="→"

# Output functions
print_banner() {
    echo ""
    echo -e "${RED}${BOLD}╔════════════════════════════════════════════════════════════╗${RESET}"
    echo -e "${RED}${BOLD}║              FlyDB Uninstallation Script v${SCRIPT_VERSION}          ║${RESET}"
    echo -e "${RED}${BOLD}╚════════════════════════════════════════════════════════════╝${RESET}"
    echo ""
}

print_step() { echo -e "${CYAN}${BOLD}==> $1${RESET}"; }
print_substep() { echo -e "    ${ICON_ARROW} $1"; }
print_success() { echo -e "${GREEN}${ICON_SUCCESS}${RESET} ${GREEN}$1${RESET}"; }
print_error() { echo -e "${RED}${ICON_ERROR}${RESET} ${RED}$1${RESET}" >&2; }
print_warning() { echo -e "${YELLOW}${ICON_WARNING}${RESET} ${YELLOW}$1${RESET}"; }
print_info() { echo -e "${BLUE}${ICON_INFO}${RESET} ${BLUE}$1${RESET}"; }
separator() { printf '%*s\n' "${1:-60}" '' | tr ' ' '─'; }

# Spinner
SPINNER_PID=""
SPINNER_MSG=""
spinner_frames=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")

spinner_start() {
    SPINNER_MSG="$1"
    if [[ ! -t 1 ]]; then echo "$SPINNER_MSG..."; return; fi
    (
        set +e; local i=0
        while true; do
            printf "\r${CYAN}${spinner_frames[$i]}${RESET} %s" "$SPINNER_MSG"
            i=$(( (i + 1) % ${#spinner_frames[@]} )); sleep 0.1
        done
    ) &
    SPINNER_PID=$!
    disown "$SPINNER_PID" 2>/dev/null || true
}

spinner_stop() {
    if [[ -n "$SPINNER_PID" ]] && kill -0 "$SPINNER_PID" 2>/dev/null; then
        kill "$SPINNER_PID" 2>/dev/null || true
        wait "$SPINNER_PID" 2>/dev/null || true
        printf "\r%*s\r" $((${#SPINNER_MSG} + 5)) ""
    fi
    SPINNER_PID=""
}

spinner_success() { spinner_stop; echo -e "${GREEN}${ICON_SUCCESS}${RESET} $1"; }
spinner_error() { spinner_stop; echo -e "${RED}${ICON_ERROR}${RESET} $1"; }

prompt_yes_no() {
    local prompt_text="$1"
    local default="${2:-y}"
    local result=""

    spinner_stop

    local hint
    if [[ "$default" == "y" ]]; then
        hint="Y/n"
    else
        hint="y/N"
    fi

    echo -en "${BOLD}${prompt_text}${RESET} [${CYAN}${hint}${RESET}]: "
    read -r result

    if [[ -z "$result" ]]; then
        result="$default"
    fi

    # Convert to lowercase for comparison (portable method)
    result=$(echo "$result" | tr '[:upper:]' '[:lower:]')

    case "$result" in
        y|yes) return 0 ;;
        *) return 1 ;;
    esac
}

# System detection
detect_os() {
    case "$(uname -s)" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        *)       OS="unknown" ;;
    esac
}

detect_init_system() {
    if [[ "$OS" == "darwin" ]]; then
        INIT_SYSTEM="launchd"
    elif command -v systemctl &>/dev/null; then
        INIT_SYSTEM="systemd"
    else
        INIT_SYSTEM="none"
    fi
}

# Sudo handling
get_sudo_cmd() {
    local target_path="$1"
    if [[ -w "$target_path" ]] || [[ -w "$(dirname "$target_path")" ]]; then
        echo ""
    elif [[ $EUID -eq 0 ]]; then
        echo ""
    else
        echo "sudo"
    fi
}

stop_sudo_keepalive() {
    if [[ -n "$SUDO_KEEPALIVE_PID" ]] && kill -0 "$SUDO_KEEPALIVE_PID" 2>/dev/null; then
        kill "$SUDO_KEEPALIVE_PID" 2>/dev/null || true
        wait "$SUDO_KEEPALIVE_PID" 2>/dev/null || true
        SUDO_KEEPALIVE_PID=""
    fi
}

obtain_sudo_if_needed() {
    if [[ "$SUDO_OBTAINED" == true ]] || [[ "$DRY_RUN" == true ]]; then
        return 0
    fi

    local needs_sudo=false
    local all_items=()
    [[ ${#FOUND_BINARIES[@]} -gt 0 ]] && all_items+=("${FOUND_BINARIES[@]}")
    [[ ${#FOUND_SERVICES[@]} -gt 0 ]] && all_items+=("${FOUND_SERVICES[@]}")
    [[ ${#FOUND_CONFIGS[@]} -gt 0 ]] && all_items+=("${FOUND_CONFIGS[@]}")

    for item in "${all_items[@]+"${all_items[@]}"}"; do
        if [[ -n "$item" ]] && [[ ! -w "$(dirname "$item")" ]]; then
            needs_sudo=true
            break
        fi
    done

    if [[ "$needs_sudo" == true ]] && [[ $EUID -ne 0 ]]; then
        echo ""
        print_info "Uninstallation requires elevated privileges (sudo)"
        echo ""

        if sudo -v; then
            SUDO_OBTAINED=true
            local parent_pid=$$
            (
                set +e
                while true; do
                    sleep 50
                    kill -0 "$parent_pid" 2>/dev/null || exit 0
                    sudo -n true 2>/dev/null || true
                done
            ) &
            SUDO_KEEPALIVE_PID=$!
            disown "$SUDO_KEEPALIVE_PID" 2>/dev/null || true
        else
            print_error "Failed to obtain sudo privileges"
            return 1
        fi
    fi
    return 0
}

# Discovery functions
find_installations() {
    print_step "Scanning for FlyDB installations..."
    echo ""

    # Binary locations to check
    local bin_locations=("/usr/local/bin" "/usr/bin" "$HOME/.local/bin")
    [[ -n "$PREFIX" ]] && bin_locations=("${PREFIX}/bin" "${bin_locations[@]}")

    for dir in "${bin_locations[@]}"; do
        if [[ -x "$dir/flydb" ]]; then
            FOUND_BINARIES+=("$dir/flydb")
        fi
        if [[ -x "$dir/fly-cli" ]]; then
            FOUND_BINARIES+=("$dir/fly-cli")
        fi
    done

    # Service files
    if [[ "$INIT_SYSTEM" == "systemd" ]]; then
        if [[ -f "/etc/systemd/system/flydb.service" ]]; then
            FOUND_SERVICES+=("/etc/systemd/system/flydb.service")
        fi
    elif [[ "$INIT_SYSTEM" == "launchd" ]]; then
        if [[ -f "/Library/LaunchDaemons/io.flydb.flydb.plist" ]]; then
            FOUND_SERVICES+=("/Library/LaunchDaemons/io.flydb.flydb.plist")
        fi
        if [[ -f "$HOME/Library/LaunchAgents/io.flydb.flydb.plist" ]]; then
            FOUND_SERVICES+=("$HOME/Library/LaunchAgents/io.flydb.flydb.plist")
        fi
    fi

    # Config directories
    if [[ -d "/etc/flydb" ]]; then
        FOUND_CONFIGS+=("/etc/flydb")
    fi
    if [[ -d "$HOME/.config/flydb" ]]; then
        FOUND_CONFIGS+=("$HOME/.config/flydb")
    fi

    # Data directories
    if [[ -d "/var/lib/flydb" ]]; then
        FOUND_DATA+=("/var/lib/flydb")
    fi
    if [[ -d "$HOME/.local/share/flydb" ]]; then
        FOUND_DATA+=("$HOME/.local/share/flydb")
    fi
}

print_found_items() {
    local total=$((${#FOUND_BINARIES[@]} + ${#FOUND_SERVICES[@]} + ${#FOUND_CONFIGS[@]} + ${#FOUND_DATA[@]}))

    if [[ $total -eq 0 ]]; then
        print_info "No FlyDB installation found"
        return 1
    fi

    echo -e "${BOLD}Found the following FlyDB components:${RESET}"
    echo ""

    if [[ ${#FOUND_BINARIES[@]} -gt 0 ]]; then
        echo -e "  ${CYAN}Binaries:${RESET}"
        for item in "${FOUND_BINARIES[@]}"; do
            echo -e "    ${ICON_ARROW} $item"
        done
    fi

    if [[ ${#FOUND_SERVICES[@]} -gt 0 ]]; then
        echo -e "  ${CYAN}Services:${RESET}"
        for item in "${FOUND_SERVICES[@]}"; do
            echo -e "    ${ICON_ARROW} $item"
        done
    fi

    if [[ ${#FOUND_CONFIGS[@]} -gt 0 ]] && [[ "$REMOVE_CONFIG" == true ]]; then
        echo -e "  ${CYAN}Configuration:${RESET}"
        for item in "${FOUND_CONFIGS[@]}"; do
            echo -e "    ${ICON_ARROW} $item"
        done
    fi

    if [[ ${#FOUND_DATA[@]} -gt 0 ]] && [[ "$REMOVE_DATA" == true ]]; then
        echo -e "  ${YELLOW}Data directories:${RESET}"
        for item in "${FOUND_DATA[@]}"; do
            echo -e "    ${ICON_ARROW} $item ${YELLOW}(contains database files!)${RESET}"
        done
    fi

    echo ""
    return 0
}

# Removal functions
remove_binaries() {
    if [[ ${#FOUND_BINARIES[@]} -eq 0 ]]; then return 0; fi

    print_step "Removing binaries..."

    for binary in "${FOUND_BINARIES[@]}"; do
        local sudo_cmd
        sudo_cmd=$(get_sudo_cmd "$binary")

        if [[ "$DRY_RUN" == true ]]; then
            print_substep "[DRY RUN] Would remove: $binary"
        else
            spinner_start "Removing $(basename "$binary")"
            if $sudo_cmd rm -f "$binary" 2>/dev/null; then
                spinner_success "Removed $binary"
            else
                spinner_error "Failed to remove $binary"
            fi
        fi
    done
    echo ""
}

remove_services() {
    if [[ ${#FOUND_SERVICES[@]} -eq 0 ]]; then return 0; fi

    print_step "Removing services..."

    for service in "${FOUND_SERVICES[@]}"; do
        local sudo_cmd
        sudo_cmd=$(get_sudo_cmd "$service")

        if [[ "$DRY_RUN" == true ]]; then
            print_substep "[DRY RUN] Would remove: $service"
        else
            # Stop and disable service first
            if [[ "$INIT_SYSTEM" == "systemd" ]]; then
                spinner_start "Stopping systemd service"
                $sudo_cmd systemctl stop flydb 2>/dev/null || true
                $sudo_cmd systemctl disable flydb 2>/dev/null || true
                spinner_success "Stopped systemd service"
            elif [[ "$INIT_SYSTEM" == "launchd" ]]; then
                spinner_start "Unloading launchd service"
                if [[ "$service" == *"LaunchDaemons"* ]]; then
                    sudo launchctl unload "$service" 2>/dev/null || true
                else
                    launchctl unload "$service" 2>/dev/null || true
                fi
                spinner_success "Unloaded launchd service"
            fi

            spinner_start "Removing service file"
            if $sudo_cmd rm -f "$service" 2>/dev/null; then
                spinner_success "Removed $service"
            else
                spinner_error "Failed to remove $service"
            fi

            # Reload systemd if needed
            if [[ "$INIT_SYSTEM" == "systemd" ]]; then
                $sudo_cmd systemctl daemon-reload 2>/dev/null || true
            fi
        fi
    done
    echo ""
}

remove_configs() {
    if [[ ${#FOUND_CONFIGS[@]} -eq 0 ]] || [[ "$REMOVE_CONFIG" != true ]]; then return 0; fi

    print_step "Removing configuration..."

    for config in "${FOUND_CONFIGS[@]}"; do
        local sudo_cmd
        sudo_cmd=$(get_sudo_cmd "$config")

        if [[ "$DRY_RUN" == true ]]; then
            print_substep "[DRY RUN] Would remove: $config"
        else
            spinner_start "Removing configuration"
            if $sudo_cmd rm -rf "$config" 2>/dev/null; then
                spinner_success "Removed $config"
            else
                spinner_error "Failed to remove $config"
            fi
        fi
    done
    echo ""
}

remove_data() {
    if [[ ${#FOUND_DATA[@]} -eq 0 ]] || [[ "$REMOVE_DATA" != true ]]; then return 0; fi

    print_step "Removing data directories..."
    print_warning "This will permanently delete all FlyDB data!"

    for data_dir in "${FOUND_DATA[@]}"; do
        local sudo_cmd
        sudo_cmd=$(get_sudo_cmd "$data_dir")

        if [[ "$DRY_RUN" == true ]]; then
            print_substep "[DRY RUN] Would remove: $data_dir"
        else
            spinner_start "Removing data"
            if $sudo_cmd rm -rf "$data_dir" 2>/dev/null; then
                spinner_success "Removed $data_dir"
            else
                spinner_error "Failed to remove $data_dir"
            fi
        fi
    done
    echo ""
}

# Argument parsing
parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --prefix)
                PREFIX="$2"
                shift 2
                ;;
            -y|--yes)
                AUTO_CONFIRM=true
                shift
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --remove-data)
                REMOVE_DATA=true
                shift
                ;;
            --no-config)
                REMOVE_CONFIG=false
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done
}

show_help() {
    cat << EOF
FlyDB Uninstallation Script v${SCRIPT_VERSION}

Usage: ./uninstall.sh [OPTIONS]

Options:
  --prefix PATH     Look for installation in specific prefix
  -y, --yes         Non-interactive mode, assume yes to all prompts
  --dry-run         Show what would be removed without removing anything
  --remove-data     Also remove data directories (WARNING: deletes databases!)
  --no-config       Don't remove configuration files
  -v, --verbose     Show verbose output
  -h, --help        Show this help message

Examples:
  ./uninstall.sh                    # Interactive uninstallation
  ./uninstall.sh --yes              # Non-interactive, remove all
  ./uninstall.sh --dry-run          # Preview what would be removed
  ./uninstall.sh --prefix ~/.local  # Uninstall from specific location
EOF
}

print_completion() {
    echo ""
    if [[ "$DRY_RUN" == true ]]; then
        echo -e "${YELLOW}${BOLD}╔════════════════════════════════════════════════════════════╗${RESET}"
        echo -e "${YELLOW}${BOLD}║              Dry Run Complete                              ║${RESET}"
        echo -e "${YELLOW}${BOLD}╚════════════════════════════════════════════════════════════╝${RESET}"
        echo ""
        print_info "No files were actually removed. Run without --dry-run to uninstall."
    else
        echo -e "${GREEN}${BOLD}╔════════════════════════════════════════════════════════════╗${RESET}"
        echo -e "${GREEN}${BOLD}║              Uninstallation Complete                       ║${RESET}"
        echo -e "${GREEN}${BOLD}╚════════════════════════════════════════════════════════════╝${RESET}"
        echo ""
        print_success "FlyDB has been successfully uninstalled"

        if [[ ${#FOUND_DATA[@]} -gt 0 ]] && [[ "$REMOVE_DATA" != true ]]; then
            echo ""
            print_info "Data directories were preserved. To remove them, run:"
            echo -e "    ${CYAN}./uninstall.sh --remove-data --yes${RESET}"
        fi
    fi
    echo ""
}

cleanup() {
    local exit_code=$?
    spinner_stop
    stop_sudo_keepalive
}

main() {
    trap cleanup EXIT
    trap 'echo ""; print_warning "Uninstallation cancelled"; exit 130' INT TERM

    print_banner
    parse_args "$@"

    if [[ "$DRY_RUN" == true ]]; then
        print_warning "DRY RUN MODE - No files will be removed"
        echo ""
    fi

    detect_os
    detect_init_system
    find_installations

    if ! print_found_items; then
        exit 0
    fi

    if [[ "$AUTO_CONFIRM" != true ]]; then
        if ! prompt_yes_no "Proceed with uninstallation?"; then
            print_info "Uninstallation cancelled"
            exit 0
        fi
        echo ""
    fi

    if ! obtain_sudo_if_needed; then
        exit 1
    fi

    remove_services
    remove_binaries
    remove_configs
    remove_data

    print_completion
}

main "$@"

