#!/bin/bash
# POTUS Installation Script
# Supports: macOS, Linux (x86_64, arm64)

set -e

VERSION="${POTUS_VERSION:-latest}"
INSTALL_DIR="${POTUS_INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="potus"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

success() {
    echo -e "${GREEN}✓${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
    exit 1
}

warn() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case "$os" in
        darwin)
            OS="Darwin"
            ;;
        linux)
            OS="Linux"
            ;;
        *)
            error "Unsupported operating system: $os"
            ;;
    esac

    case "$arch" in
        x86_64|amd64)
            ARCH="x86_64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        *)
            error "Unsupported architecture: $arch"
            ;;
    esac

    info "Detected platform: $OS $ARCH"
}

# Get latest version from GitHub
get_latest_version() {
    if [ "$VERSION" = "latest" ]; then
        info "Fetching latest version..."
        VERSION=$(curl -sL https://api.github.com/repos/taaha3244/potus/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

        if [ -z "$VERSION" ]; then
            error "Failed to fetch latest version"
        fi

        success "Latest version: $VERSION"
    fi
}

# Download and install
install_potus() {
    local filename="potus_${VERSION}_${OS}_${ARCH}.tar.gz"
    local download_url="https://github.com/taaha3244/potus/releases/download/${VERSION}/${filename}"
    local tmp_dir=$(mktemp -d)

    info "Downloading POTUS $VERSION..."
    if ! curl -sL "$download_url" -o "$tmp_dir/$filename"; then
        error "Failed to download POTUS from $download_url"
    fi

    info "Extracting archive..."
    tar -xzf "$tmp_dir/$filename" -C "$tmp_dir"

    info "Installing to $INSTALL_DIR..."
    if [ ! -w "$INSTALL_DIR" ]; then
        warn "Need sudo privileges to install to $INSTALL_DIR"
        sudo install -m 755 "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    else
        install -m 755 "$tmp_dir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    fi

    # Cleanup
    rm -rf "$tmp_dir"

    success "POTUS installed successfully!"
}

# Verify installation
verify_installation() {
    info "Verifying installation..."

    if ! command -v potus &> /dev/null; then
        error "POTUS binary not found in PATH. Please add $INSTALL_DIR to your PATH."
    fi

    local installed_version=$(potus --version 2>&1 | head -n1)
    success "Installed: $installed_version"
}

# Post-installation instructions
show_next_steps() {
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  POTUS Installation Complete!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "1. Set up your API key:"
    echo "   ${BLUE}export ANTHROPIC_API_KEY=\"your-api-key\"${NC}"
    echo "   ${BLUE}export OPENAI_API_KEY=\"your-api-key\"${NC}"
    echo ""
    echo "2. Run POTUS:"
    echo "   ${BLUE}potus${NC}"
    echo ""
    echo "3. Get help:"
    echo "   ${BLUE}potus --help${NC}"
    echo ""
    echo "4. Read the quick start guide:"
    echo "   ${BLUE}https://github.com/taaha3244/potus/blob/main/QUICKSTART.md${NC}"
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Main installation flow
main() {
    echo ""
    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                                                           ║${NC}"
    echo -e "${BLUE}║           POTUS - Power Of The Universal Shell            ║${NC}"
    echo -e "${BLUE}║              AI Coding Agent Installation                 ║${NC}"
    echo -e "${BLUE}║                                                           ║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo ""

    detect_platform
    get_latest_version
    install_potus
    verify_installation
    show_next_steps
}

main "$@"
