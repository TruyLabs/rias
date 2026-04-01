# Release & Distribution Guide

This document explains how to build and release kai binaries to the `tinhvqbk/kai` repository.

## Architecture

- **`tinhvqbk/kai-code`** (this repo) — source code and build configuration
- **`tinhvqbk/kai`** — pre-built binaries and releases

## Building & Releasing

### 1. Manual Trigger Build

Trigger a new build via GitHub Actions:

1. Go to **tinhvqbk/kai-code** → Actions → **Build and Release** workflow
2. Click **Run workflow**
3. Enter version tag (e.g., `v1.0.0`)
4. Click **Run workflow**

The workflow will:
- Build for macOS (Intel + Apple Silicon) and Linux
- Create checksums for each binary
- Push releases to `tinhvqbk/kai`

### 2. Manual Build (Local)

If you prefer to build locally:

```bash
# Build all platforms
make build GOOS=darwin GOARCH=amd64  # macOS Intel
make build GOOS=darwin GOARCH=arm64  # macOS Apple Silicon
make build GOOS=linux GOARCH=amd64   # Linux

# Create checksums
sha256sum kai-darwin-amd64 > kai-darwin-amd64.sha256
sha256sum kai-darwin-arm64 > kai-darwin-arm64.sha256
sha256sum kai-linux-amd64 > kai-linux-amd64.sha256

# Create GitHub Release manually
# Upload binaries and checksums to tinhvqbk/kai release page
```

## User Installation Methods

### 1. Shell Script (Recommended for most users)

```bash
curl -fsSL https://raw.githubusercontent.com/tinhvqbk/kai/main/install.sh | bash
```

Features:
- Auto-detects platform (macOS Intel/Apple Silicon, Linux)
- Verifies checksums
- Installs to `/usr/local/bin/kai`
- Prompts for sudo if needed

### 2. Direct Download

```bash
# Get the latest release
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-amd64 -o /usr/local/bin/kai
chmod +x /usr/local/bin/kai
```

### 3. Homebrew (Setup Required)

To enable `brew install tinhvqbk/kai/kai`:

#### Option A: Use a Homebrew Tap Repository

1. Create a new repo: `homebrew-kai`
2. Create `Formula/kai.rb` (see example below)
3. Users run:
```bash
brew tap tinhvqbk/kai
brew install kai
```

#### Option B: Embed in Main Repo

1. Keep `Formula/kai.rb` in this repo
2. Users run:
```bash
brew install tinhvqbk/kai-code@Formula/kai.rb
```

#### Setting up a Homebrew Tap

Create a new GitHub repo `homebrew-kai`:

```bash
git clone https://github.com/tinhvqbk/homebrew-kai.git
cd homebrew-kai
mkdir -p Formula
# Copy Formula/kai.rb from this repo
# Update SHA256 hashes with actual values from release
git add .
git commit -m "Add kai formula"
git push
```

Users then install via:
```bash
brew tap tinhvqbk/homebrew-kai
brew install kai
```

### 4. From Source

```bash
git clone https://github.com/tinhvqbk/kai-code.git
cd kai-code
make build
./kai
```

### 5. Go Install

```bash
go install github.com/tinhvqbk/kai/cmd/kai@latest
```

## Updating Homebrew Formula

After each release, update the SHA256 hashes in `Formula/kai.rb`:

```bash
# Get checksums from GitHub release
curl -s https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-amd64.sha256 | cut -d' ' -f1
curl -s https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-arm64.sha256 | cut -d' ' -f1
curl -s https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64.sha256 | cut -d' ' -f1
```

Then update `Formula/kai.rb` with the new hashes and version.

## Release Checklist

- [ ] Update version in code if needed
- [ ] Create git tag: `git tag v1.0.0`
- [ ] Push tag: `git push origin v1.0.0`
- [ ] Trigger GitHub Actions workflow with version `v1.0.0`
- [ ] Verify binaries built in `tinhvqbk/kai` releases
- [ ] Verify checksums match
- [ ] Test installation via:
  - [ ] `curl ... | bash` script
  - [ ] Direct download and `chmod +x`
  - [ ] `go install` (if applicable)
- [ ] Update Homebrew formula with new SHA256 hashes
- [ ] Announce release

## Troubleshooting

### Workflow fails to authenticate

Ensure the workflow has permission to write to `tinhvqbk/kai`:
1. Go to `tinhvqbk/kai-code` → Settings → Actions → General
2. Set "Workflow permissions" to "Read and write permissions"

### Binaries fail signature/checksum

Ensure release workflow is using correct ldflags with version from git tag.

### Install script not working

Test manually:
```bash
curl -fsSL https://raw.githubusercontent.com/tinhvqbk/kai/main/install.sh
# Review output, then pipe to bash
```

## References

- [GitHub Actions: Create Release](https://github.com/softprops/action-gh-release)
- [Homebrew Formula Documentation](https://docs.brew.sh/Formula-Cookbook)
- [Go Build Flags](https://golang.org/cmd/go/#hdr-Compile_packages_and_dependencies)
