#!/bin/bash
set -eo pipefail

# kai installation script
# Downloads the latest pre-built binary for your platform
# Usage: curl -fsSL https://raw.githubusercontent.com/norenis/kai/main/install.sh | bash
#        curl -fsSL ... | bash -s -- --version v1.2.3

REPO="norenis/kai"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
GITHUB_API="https://api.github.com/repos/$REPO/releases"
VERSION=""  # Set by --version flag or auto-detected

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Temp files — cleaned up on exit
TEMP_BINARY=""
TEMP_CHECKSUM=""
cleanup() {
  [ -n "$TEMP_BINARY" ]   && rm -f "$TEMP_BINARY"
  [ -n "$TEMP_CHECKSUM" ] && rm -f "$TEMP_CHECKSUM"
}
trap cleanup EXIT

# ── Argument parsing ────────────────────────────────────────────────────────

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --version)
        VERSION="$2"
        shift 2
        ;;
      --install-dir)
        INSTALL_DIR="$2"
        shift 2
        ;;
      --help|-h)
        echo "Usage: install.sh [--version v1.2.3] [--install-dir /usr/local/bin]"
        exit 0
        ;;
      *)
        echo -e "${RED}Unknown argument: $1${NC}"
        exit 1
        ;;
    esac
  done
}

# ── Platform detection ───────────────────────────────────────────────────────

detect_platform() {
  local OS ARCH

  OS="$(uname -s)"
  ARCH="$(uname -m)"

  case "$OS" in
    Darwin) OS_TYPE="darwin" ;;
    Linux)  OS_TYPE="linux"  ;;
    *)
      echo -e "${RED}Error: Unsupported OS: $OS${NC}"
      exit 1
      ;;
  esac

  case "$ARCH" in
    x86_64)       ARCH_TYPE="amd64" ;;
    arm64|aarch64) ARCH_TYPE="arm64" ;;
    *)
      echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
      exit 1
      ;;
  esac

  # linux-arm64 is not currently built — fail early with a clear message.
  if [ "$OS_TYPE" = "linux" ] && [ "$ARCH_TYPE" = "arm64" ]; then
    echo -e "${RED}Error: linux/arm64 builds are not yet available.${NC}"
    echo "Please build from source: https://github.com/$REPO#contributing"
    exit 1
  fi

  BINARY_NAME="kai-${OS_TYPE}-${ARCH_TYPE}"
  echo -e "${YELLOW}Platform: $OS_TYPE/$ARCH_TYPE${NC}"
}

# ── Version resolution ───────────────────────────────────────────────────────

get_version() {
  if ! command -v curl &>/dev/null; then
    echo -e "${RED}Error: curl is required but not installed${NC}"
    exit 1
  fi

  if [ -n "$VERSION" ]; then
    echo -e "${YELLOW}Requested version: $VERSION${NC}"
    return
  fi

  local response
  response="$(curl -sf "$GITHUB_API/latest")"

  if echo "$response" | grep -q "API rate limit exceeded"; then
    echo -e "${RED}Error: GitHub API rate limit exceeded. Try again later or use --version.${NC}"
    exit 1
  fi

  VERSION="$(echo "$response" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4)"

  if [ -z "$VERSION" ]; then
    echo -e "${RED}Error: Could not determine latest version${NC}"
    exit 1
  fi

  echo -e "${YELLOW}Latest version: $VERSION${NC}"
}

# ── Upgrade check ─────────────────────────────────────────────────────────────

check_existing() {
  local existing_bin="$INSTALL_DIR/kai"
  if [ ! -x "$existing_bin" ]; then
    return
  fi

  local installed_version
  installed_version="$("$existing_bin" version 2>/dev/null | awk '{print $1}' || echo "")"

  if [ "$installed_version" = "$VERSION" ]; then
    echo -e "${GREEN}kai $VERSION is already installed and up-to-date.${NC}"
    exit 0
  fi

  if [ -n "$installed_version" ]; then
    echo -e "${YELLOW}Upgrading kai $installed_version → $VERSION${NC}"
  fi
}

# ── Download ─────────────────────────────────────────────────────────────────

download_binary() {
  local download_url="https://github.com/$REPO/releases/download/$VERSION/$BINARY_NAME"
  TEMP_BINARY="$(mktemp /tmp/kai.XXXXXX)"

  echo -e "${YELLOW}Downloading $BINARY_NAME ($VERSION)...${NC}"

  if ! curl -fL --progress-bar -o "$TEMP_BINARY" "$download_url"; then
    echo -e "${RED}Error: Download failed. Check that $VERSION exists for $BINARY_NAME.${NC}"
    exit 1
  fi

  chmod +x "$TEMP_BINARY"
  echo -e "${GREEN}✓ Downloaded${NC}"
}

# ── Checksum verification ─────────────────────────────────────────────────────

verify_checksum() {
  local checksum_url="https://github.com/$REPO/releases/download/$VERSION/$BINARY_NAME.sha256"
  TEMP_CHECKSUM="$(mktemp /tmp/kai-checksum.XXXXXX)"

  echo -e "${YELLOW}Verifying checksum...${NC}"

  if ! curl -fLs -o "$TEMP_CHECKSUM" "$checksum_url" 2>/dev/null; then
    echo -e "${YELLOW}⚠ Checksum file not available — skipping verification${NC}"
    return
  fi

  # Rewrite checksum file to reference the temp binary path.
  local expected_hash
  expected_hash="$(awk '{print $1}' "$TEMP_CHECKSUM")"
  echo "$expected_hash  $TEMP_BINARY" > "$TEMP_CHECKSUM"

  if command -v sha256sum &>/dev/null; then
    if ! sha256sum -c "$TEMP_CHECKSUM" &>/dev/null; then
      echo -e "${RED}Error: Checksum verification failed${NC}"
      exit 1
    fi
  elif command -v shasum &>/dev/null; then
    if ! shasum -a 256 -c "$TEMP_CHECKSUM" &>/dev/null; then
      echo -e "${RED}Error: Checksum verification failed${NC}"
      exit 1
    fi
  else
    echo -e "${YELLOW}⚠ No sha256sum or shasum found — skipping verification${NC}"
    return
  fi

  echo -e "${GREEN}✓ Checksum verified${NC}"
}

# ── Install ───────────────────────────────────────────────────────────────────

install_binary() {
  if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "${YELLOW}Creating $INSTALL_DIR${NC}"
    mkdir -p "$INSTALL_DIR" 2>/dev/null || sudo mkdir -p "$INSTALL_DIR"
  fi

  if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_BINARY" "$INSTALL_DIR/kai"
  else
    echo -e "${YELLOW}Installing to $INSTALL_DIR requires sudo${NC}"
    sudo mv "$TEMP_BINARY" "$INSTALL_DIR/kai"
    sudo chmod +x "$INSTALL_DIR/kai"
  fi
  TEMP_BINARY=""  # Moved — don't try to clean it up.

  echo -e "${GREEN}✓ Installed to $INSTALL_DIR/kai${NC}"
}

# ── Post-install ──────────────────────────────────────────────────────────────

verify_installation() {
  # Use the direct path — $INSTALL_DIR may not be in PATH yet.
  local installed_version
  installed_version="$("$INSTALL_DIR/kai" version 2>/dev/null | head -1 || echo "unknown")"
  echo -e "${GREEN}✓ $installed_version${NC}"

  if ! command -v kai &>/dev/null; then
    echo ""
    echo -e "${YELLOW}⚠ '$INSTALL_DIR' is not in your PATH.${NC}"
    echo "Add it to your ~/.bashrc or ~/.zshrc:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
  fi

  echo ""
  echo "Next steps:"
  echo "  kai setup                        # initialize ~/.kai/ and register MCP server"
  echo "  kai auth set-key --provider claude"
  echo "  kai                              # start chatting"
  echo ""
  echo "Run 'kai --help' to see all commands."
}

# ── Main ──────────────────────────────────────────────────────────────────────

main() {
  echo -e "${GREEN}=== kai Installation ===${NC}"
  echo ""

  parse_args "$@"
  detect_platform
  get_version
  check_existing
  download_binary
  verify_checksum
  install_binary
  verify_installation

  echo ""
  echo -e "${GREEN}=== Done ===${NC}"
}

main "$@"
