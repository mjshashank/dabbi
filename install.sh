#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="mjshashank/dabbi"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="dabbi"

echo -e "${BLUE}"
echo "     _       _     _     _ "
echo "  __| | __ _| |__ | |__ (_)"
echo " / _\` |/ _\` | '_ \| '_ \| |"
echo "| (_| | (_| | |_) | |_) | |"
echo " \__,_|\__,_|_.__/|_.__/|_|"
echo -e "${NC}"
echo ""

# Check for required tools
for cmd in curl tar; do
    if ! command -v $cmd &> /dev/null; then
        echo -e "${RED}Error: $cmd is required but not installed.${NC}"
        exit 1
    fi
done

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    darwin|linux)
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        echo "Please install manually from https://github.com/$REPO"
        exit 1
        ;;
esac

echo -e "${BLUE}Detected:${NC} $OS/$ARCH"
echo ""

# Check for multipass
check_multipass() {
    if command -v multipass &> /dev/null; then
        echo -e "${GREEN}[OK]${NC} multipass is installed"
        return 0
    else
        return 1
    fi
}

# Install multipass
install_multipass() {
    echo -e "${YELLOW}[INFO]${NC} Installing multipass..."

    case "$OS" in
        darwin)
            if command -v brew &> /dev/null; then
                echo "Installing via Homebrew..."
                brew install --cask multipass
            else
                echo -e "${RED}Error: Homebrew not found.${NC}"
                echo "Please install multipass manually:"
                echo "  1. Install Homebrew: https://brew.sh"
                echo "  2. Run: brew install --cask multipass"
                echo ""
                echo "Or download directly from: https://multipass.run/download/macos"
                exit 1
            fi
            ;;
        linux)
            if command -v snap &> /dev/null; then
                echo "Installing via snap..."
                sudo snap install multipass
            else
                echo -e "${RED}Error: snap not found.${NC}"
                echo "Please install multipass manually:"
                echo "  1. Install snap: sudo apt install snapd"
                echo "  2. Run: sudo snap install multipass"
                echo ""
                echo "Or see: https://multipass.run/install"
                exit 1
            fi
            ;;
    esac

    echo -e "${GREEN}[OK]${NC} multipass installed"
}

# Check and install multipass if needed
echo -e "${BLUE}Checking dependencies...${NC}"
if ! check_multipass; then
    echo ""
    read -p "multipass is required but not installed. Install it now? [Y/n] " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Nn]$ ]]; then
        echo -e "${YELLOW}[WARN]${NC} Skipping multipass installation."
        echo "dabbi will not work without multipass."
    else
        install_multipass
    fi
fi
echo ""

# Get latest release version
echo -e "${BLUE}Fetching latest release...${NC}"
LATEST_VERSION=$(curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo -e "${YELLOW}[WARN]${NC} Could not fetch latest version, using 'latest'"
    DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/dabbi-$OS-$ARCH.tar.gz"
else
    echo -e "${GREEN}[OK]${NC} Latest version: $LATEST_VERSION"
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/dabbi-$OS-$ARCH.tar.gz"
fi
echo ""

# Download and install
echo -e "${BLUE}Downloading dabbi...${NC}"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/dabbi.tar.gz"; then
    echo -e "${RED}Error: Failed to download dabbi${NC}"
    echo "URL: $DOWNLOAD_URL"
    echo ""
    echo "You can build from source instead:"
    echo "  git clone https://github.com/$REPO"
    echo "  cd dabbi && make install"
    exit 1
fi

echo -e "${GREEN}[OK]${NC} Downloaded"
echo ""

# Extract
echo -e "${BLUE}Extracting...${NC}"
tar -xzf "$TMP_DIR/dabbi.tar.gz" -C "$TMP_DIR"
echo -e "${GREEN}[OK]${NC} Extracted"
echo ""

# Install binary
echo -e "${BLUE}Installing to $INSTALL_DIR...${NC}"
if [ -w "$INSTALL_DIR" ]; then
    cp "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Requires sudo to install to $INSTALL_DIR"
    sudo cp "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
    sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
fi
echo -e "${GREEN}[OK]${NC} Installed to $INSTALL_DIR/$BINARY_NAME"
echo ""

# Verify installation
if command -v dabbi &> /dev/null; then
    VERSION=$(dabbi --version 2>/dev/null || echo "unknown")
    echo -e "${GREEN}Successfully installed dabbi${NC}"
    echo ""
else
    echo -e "${YELLOW}[WARN]${NC} dabbi installed but not in PATH"
    echo "Add $INSTALL_DIR to your PATH, or run directly: $INSTALL_DIR/$BINARY_NAME"
    echo ""
fi

# Print next steps
echo -e "${BLUE}Next steps:${NC}"
echo ""
echo "  1. Start the daemon (port 80 requires sudo):"
echo "     ${GREEN}sudo dabbi serve${NC}"
echo ""
echo "     Or on a different port:"
echo "     ${GREEN}dabbi serve --port 8080${NC}"
echo ""
echo "  2. Open the web UI:"
echo "     ${GREEN}http://localhost${NC}"
echo ""
echo "  3. Create your first VM:"
echo "     ${GREEN}dabbi create dev${NC}"
echo ""
echo "  4. Open a shell:"
echo "     ${GREEN}dabbi shell dev${NC}"
echo ""
echo "Your auth token is stored in ~/.dabbi/config.json"
echo ""
echo "For more info, run: ${GREEN}dabbi --help${NC}"
echo ""
