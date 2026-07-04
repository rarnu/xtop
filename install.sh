#!/bin/sh
# One-liner installer for xtop.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | bash
#   curl -fsSL ... | bash -s -- --uninstall
# Environment variables:
#   XTOP_VERSION    Specific version to install (e.g. 0.1.0). Default: latest.
#   XTOP_INSTALL_DIR Directory to install the binary. Default: /usr/local/bin or ~/.local/bin.

set -eu

REPO="rarnu/xtop"
BINARY="xtop"

# Print helpers
info() { printf '\033[34mINFO\033[0m %s\n' "$*"; }
warn() { printf '\033[33mWARN\033[0m %s\n' "$*" >&2; }
err() { printf '\033[31mERR \033[0m %s\n' "$*" >&2; }

# Detect OS
 detect_os() {
    case "$(uname -s)" in
        Linux*) echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *) echo "unsupported" ;;
    esac
}

# Detect architecture
 detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

# Resolve latest release version from GitHub API
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        awk -F'"' '/"tag_name":/ {print $4}' | sed 's/^v//'
}

# Download URL for a given version/os/arch
get_download_url() {
    version="$1"
    os="$2"
    arch="$3"
    echo "https://github.com/${REPO}/releases/download/v${version}/${BINARY}_${version}_${os}_${arch}.tar.gz"
}

# Prefer sudo if available and we are not root
maybe_sudo() {
    if [ "$(id -u)" -eq 0 ]; then
        "$@"
    elif command -v sudo >/dev/null 2>&1; then
        sudo "$@"
    else
        "$@"
    fi
}

# Install language files to system or user directory
install_lang() {
    src_dir="$1"
    if [ -d "${src_dir}/lang" ]; then
        if maybe_sudo install -d -m 755 "/etc/xtop/lang" 2>/dev/null; then
            maybe_sudo cp -R "${src_dir}/lang/"* "/etc/xtop/lang/"
            info "Language files installed to /etc/xtop/lang"
        else
            mkdir -p "${HOME}/.xtop/lang"
            cp -R "${src_dir}/lang/"* "${HOME}/.xtop/lang/"
            info "Language files installed to ~/.xtop/lang"
        fi
    fi
}

# Remove language files
uninstall_lang() {
    if [ -d "/etc/xtop/lang" ]; then
        maybe_sudo rm -rf "/etc/xtop/lang"
        info "Removed /etc/xtop/lang"
    fi
    if [ -d "${HOME}/.xtop/lang" ]; then
        rm -rf "${HOME}/.xtop/lang"
        info "Removed ~/.xtop/lang"
    fi
}

# Install xtop
do_install() {
    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ "$os" = "unsupported" ]; then
        err "Unsupported operating system. xtop supports Linux and macOS."
        exit 1
    fi
    if [ "$arch" = "unsupported" ]; then
        err "Unsupported architecture: $(uname -m). xtop supports amd64 and arm64."
        exit 1
    fi
    if [ "$os" = "darwin" ] && [ "$arch" = "amd64" ]; then
        err "Unsupported platform: darwin/amd64. xtop only supports darwin/arm64 (Apple Silicon)."
        exit 1
    fi

    version="${XTOP_VERSION:-$(get_latest_version)}"
    if [ -z "$version" ]; then
        err "Could not determine the latest version. Set XTOP_VERSION manually."
        exit 1
    fi

    install_dir="${XTOP_INSTALL_DIR:-}"
    if [ -z "$install_dir" ]; then
        if [ -w "/usr/local/bin" ] || command -v sudo >/dev/null 2>&1; then
            install_dir="/usr/local/bin"
        else
            install_dir="${HOME}/.local/bin"
        fi
    fi

    if [ ! -d "$install_dir" ]; then
        mkdir -p "$install_dir" || {
            err "Failed to create install directory: $install_dir"
            exit 1
        }
    fi

    target="${install_dir}/${BINARY}"

    if [ -e "$target" ]; then
        warn "${target} already exists."
        if [ "${FORCE:-}" != "1" ]; then
            printf "Overwrite? [y/N] "
            read -r reply
            case "$reply" in
                y|Y) ;;
                *) info "Installation cancelled."; exit 0 ;;
            esac
        fi
    fi

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    url="$(get_download_url "$version" "$os" "$arch")"
    archive="${tmpdir}/${BINARY}_${version}_${os}_${arch}.tar.gz"

    info "Downloading xtop v${version} for ${os}/${arch}..."
    if ! curl -fsSL "$url" -o "$archive"; then
        err "Download failed: $url"
        exit 1
    fi

    info "Extracting..."
    tar -xzf "$archive" -C "$tmpdir"

    extracted="${tmpdir}/${BINARY}_${version}_${os}_${arch}/${BINARY}"
    if [ ! -f "$extracted" ]; then
        err "Expected binary ${BINARY} not found in archive."
        exit 1
    fi

    chmod +x "$extracted"

    info "Installing ${BINARY} to ${target}..."
    if [ -w "$install_dir" ]; then
        mv "$extracted" "$target"
    else
        maybe_sudo mv "$extracted" "$target"
    fi

    install_lang "${tmpdir}/${BINARY}_${version}_${os}_${arch}"

    if ! command -v "$BINARY" >/dev/null 2>&1; then
        case ":${PATH}:" in
            *":${install_dir}:"*) ;;
            *) warn "${install_dir} is not in your PATH. Add it to your shell profile to use '${BINARY}' directly." ;;
        esac
    fi

    info "xtop v${version} installed successfully."
    info "Run 'xtop --version' or 'xtop --help' to get started."
}

# Uninstall xtop
do_uninstall() {
    install_dir="${XTOP_INSTALL_DIR:-}"
    if [ -z "$install_dir" ]; then
        if [ -w "/usr/local/bin" ] || command -v sudo >/dev/null 2>&1; then
            install_dir="/usr/local/bin"
        else
            install_dir="${HOME}/.local/bin"
        fi
    fi
    target="${install_dir}/${BINARY}"

    if [ -f "$target" ]; then
        if [ -w "$install_dir" ]; then
            rm -f "$target"
        else
            maybe_sudo rm -f "$target"
        fi
        info "Removed ${target}"
    else
        warn "${target} not found."
    fi

    uninstall_lang
    info "Uninstall complete."
}

# Main
main() {
    case "${1:-}" in
        --uninstall|-u) do_uninstall ;;
        --help|-h)
            cat <<'EOF'
xtop installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/rarnu/xtop/main/install.sh | bash
  curl -fsSL ... | bash -s -- --uninstall

Environment variables:
  XTOP_VERSION      Version to install (default: latest GitHub release)
  XTOP_INSTALL_DIR  Directory for the binary (default: /usr/local/bin or ~/.local/bin)
  FORCE=1           Skip overwrite confirmation
EOF
            ;;
        *) do_install ;;
    esac
}

main "$@"
