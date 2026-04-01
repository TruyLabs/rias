#!/bin/bash
set -e

# kai installation script
# Downloads the latest pre-built binary for your platform

REPO="tinhvqbk/kai"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_LATEST_URL="https://api.github.com/repos/$REPO/releases/latest"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
detect_platform() {
  local OS=$(uname -s)
  local ARCH=$(uname -m)

  case "$OS" in
    Darwin)
      OS_TYPE="darwin"
      ;;
    Linux)
      OS_TYPE="linux"
      ;;
    *)
      echo -e "${RED}Error: Unsupported OS: $OS${NC}"
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64)
      ARCH_TYPE="amd64"
      ;;
    arm64|aarch64)
      ARCH_TYPE="arm64"
      ;;
    *)
      echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
      exit 1
      ;;
  esac

  BINARY_NAME="kai-${OS_TYPE}-${ARCH_TYPE}"
  echo -e "${YELLOW}Detected platform: $OS_TYPE/$ARCH_TYPE${NC}"
}

# Get the latest release version
get_latest_version() {
  if ! command -v curl &> /dev/null; then
    echo -e "${RED}Error: curl is required but not installed${NC}"
    exit 1
  fi

  local response=$(curl -s "$GITHUB_LATEST_URL")

  if echo "$response" | grep -q "API rate limit exceeded"; then
    echo -e "${RED}Error: GitHub API rate limit exceeded. Please try again later.${NC}"
    exit 1
  fi

  VERSION=$(echo "$response" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)

  if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: Could not determine latest version${NC}"
    exit 1
  fi

  echo -e "${YELLOW}Latest version: $VERSION${NC}"
}

# Download the binary
download_binary() {
  local DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_NAME"
  local TEMP_FILE="/tmp/$BINARY_NAME"

  echo -e "${YELLOW}Downloading $BINARY_NAME from $VERSION...${NC}"

  if ! curl -fL -o "$TEMP_FILE" "$DOWNLOAD_URL"; then
    echo -e "${RED}Error: Failed to download binary${NC}"
    exit 1
  fi

  if [ ! -f "$TEMP_FILE" ]; then
    echo -e "${RED}Error: Download failed - file not found${NC}"
    exit 1
  fi

  chmod +x "$TEMP_FILE"
  echo -e "${GREEN}✓ Downloaded successfully${NC}"

  TEMP_BINARY="$TEMP_FILE"
}

# Verify checksum if available
verify_checksum() {
  local CHECKSUM_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_NAME.sha256"
  local TEMP_CHECKSUM="/tmp/$BINARY_NAME.sha256"

  echo -e "${YELLOW}Verifying checksum...${NC}"

  if ! curl -fLs -o "$TEMP_CHECKSUM" "$CHECKSUM_URL" 2>/dev/null; then
    echo -e "${YELLOW}⚠ Checksum file not available (skipping verification)${NC}"
    return
  fi

  if command -v sha256sum &> /dev/null; then
    cd /tmp
    if ! sha256sum -c "$BINARY_NAME.sha256" --quiet 2>/dev/null; then
      echo -e "${RED}Error: Checksum verification failed${NC}"
      rm -f "$TEMP_BINARY" "$TEMP_CHECKSUM"
      exit 1
    fi
  elif command -v shasum &> /dev/null; then
    cd /tmp
    if ! shasum -a 256 -c "$BINARY_NAME.sha256" --quiet 2>/dev/null; then
      echo -e "${RED}Error: Checksum verification failed${NC}"
      rm -f "$TEMP_BINARY" "$TEMP_CHECKSUM"
      exit 1
    fi
  fi

  echo -e "${GREEN}✓ Checksum verified${NC}"
  rm -f "$TEMP_CHECKSUM"
}

# Install the binary
install_binary() {
  if [ ! -w "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Installing to $INSTALL_DIR requires sudo${NC}"
    sudo mv "$TEMP_BINARY" "$INSTALL_DIR/kai"
    sudo chmod +x "$INSTALL_DIR/kai"
  else
    mv "$TEMP_BINARY" "$INSTALL_DIR/kai"
    chmod +x "$INSTALL_DIR/kai"
  fi

  echo -e "${GREEN}✓ Installed to $INSTALL_DIR/kai${NC}"
}

# Verify installation
verify_installation() {
  if ! command -v kai &> /dev/null; then
    echo -e "${YELLOW}⚠ 'kai' not found in PATH. Make sure $INSTALL_DIR is in your PATH.${NC}"
    echo "Add to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    return
  fi

  local INSTALLED_VERSION=$(kai version 2>/dev/null | head -1 || echo "unknown")
  echo -e "${GREEN}✓ Installation complete!${NC}"
  echo "Installed version: $INSTALLED_VERSION"
  echo ""
  echo "Get started:"
  echo "  kai --help"
  echo "  kai ask \"What do I think about testing?\""
  echo "  kai dashboard"
}

# Main installation flow
main() {
  echo -e "${GREEN}=== kai Installation ===${NC}\n"

  detect_platform
  get_latest_version
  download_binary
  verify_checksum
  install_binary
  verify_installation

  echo -e "\n${GREEN}=== Installation successful! ===${NC}"
}

main "$@"
