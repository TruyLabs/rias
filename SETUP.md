# Production Setup Guide

This guide explains how to set up the kai distribution system on GitHub.

## Repository Structure

Two GitHub repositories are needed:

1. **tinhvqbk/kai-code** (source code)
   - Contains all source files
   - Build and release automation
   - Installation scripts

2. **tinhvqbk/kai** (binary releases)
   - Pre-built binaries only
   - GitHub Releases with checksums
   - No source code or CI/CD

## Initial Setup

### Step 1: Create Both Repositories

#### Create tinhvqbk/kai-code
```bash
# On GitHub.com create new repo "kai-code"
# - Visibility: Public
# - No README (we have one)
# - No .gitignore (we have one)
# - No license (we have one)

# Then:
git remote set-url origin https://github.com/tinhvqbk/kai-code.git
git push -u origin main
```

#### Create tinhvqbk/kai
```bash
# On GitHub.com create new repo "kai"
# - Visibility: Public
# - Initialize with README
# - No .gitignore
# - License: MIT

# This repo stores releases only (no source code)
# No need to clone locally initially
```

### Step 2: Configure GitHub Actions Permissions

**For tinhvqbk/kai-code:**

1. Go to Settings → Actions → General
2. Scroll to "Workflow permissions"
3. Select **"Read and write permissions"**
4. Check "Allow GitHub Actions to create and approve pull requests" (optional)
5. Save

**For tinhvqbk/kai:**

1. Go to Settings → Actions → General
2. Select **"Read and write permissions"**
3. Save

### Step 3: Create GITHUB_TOKEN

The workflow uses `secrets.GITHUB_TOKEN` automatically (no manual setup needed).

To verify it works:
```bash
# In tinhvqbk/kai-code repo, go to:
# Settings → Secrets and variables → Actions
# You should see "GITHUB_TOKEN" listed (it's automatic)
```

### Step 4: Test the Build Workflow

1. Go to **tinhvqbk/kai-code** → Actions
2. Select **"Build and Release"** workflow
3. Click **"Run workflow"**
4. Enter version: `v1.0.0` (or similar)
5. Click **Run workflow**
6. Monitor the build:
   - Build job (3-5 minutes total for all platforms)
   - Release job (creates the release)
   - Check **tinhvqbk/kai** releases page

Expected result: New release in tinhvqbk/kai with 6 files:
- kai-darwin-amd64
- kai-darwin-amd64.sha256
- kai-darwin-arm64
- kai-darwin-arm64.sha256
- kai-linux-amd64
- kai-linux-amd64.sha256

## Files in tinhvqbk/kai-code

The source repo should contain:

```
tinhvqbk/kai-code/
├── .github/
│   └── workflows/
│       ├── build-release.yml      # Manual trigger builds
│       └── verify-build.yml       # CI for commits
├── cmd/
│   └── kai/
│       └── main.go
├── internal/
│   └── ...
├── Formula/
│   └── kai.rb                     # Homebrew formula
├── install.sh                     # Universal install script
├── Makefile                       # Build configuration
├── README.md                      # User documentation
├── RELEASE.md                     # Release guide
├── PRODUCTION.md                  # Production checklist
├── SETUP.md                       # This file
├── go.mod
├── go.sum
└── config.yaml.example
```

## Files in tinhvqbk/kai

The releases repo can be minimal:

```
tinhvqbk/kai/
├── README.md                      # Install instructions
└── [releases created automatically via workflow]
```

### Populate tinhvqbk/kai README

Create a simple README in tinhvqbk/kai:

```markdown
# kai Releases

Pre-built binaries for kai.

## Installation

### Quick Install
\`\`\`bash
curl -fsSL https://raw.githubusercontent.com/tinhvqbk/kai/main/install.sh | bash
\`\`\`

### Manual Download
Download from [Releases](https://github.com/tinhvqbk/kai/releases)

### From Source
\`\`\`bash
git clone https://github.com/tinhvqbk/kai-code.git
cd kai-code
make build
./kai
\`\`\`

## Verification

Each release includes SHA256 checksums. Verify downloads:

\`\`\`bash
sha256sum -c kai-linux-amd64.sha256
\`\`\`

## Documentation

See [tinhvqbk/kai-code](https://github.com/tinhvqbk/kai-code) for:
- Build instructions
- Contributing guidelines
- Full documentation
```

## Continuous Integration

### On Every Push to main (in kai-code)
**Workflow: verify-build.yml**
- Runs tests
- Checks Go formatting
- Verifies install script syntax
- Scans for secrets
- Builds binary

### On Manual Trigger (Build and Release)
**Workflow: build-release.yml**
- Validates version format
- Builds for all platforms
- Creates checksums
- Creates GitHub release in kai
- Publishes binaries

## Troubleshooting

### "Permission denied" error in workflow
Solution: Check GitHub Actions permissions (see Step 2 above)

### Binaries not appearing in tinhvqbk/kai
Solution: Check workflow logs in tinhvqbk/kai-code Actions

### Build fails with "Go version mismatch"
Solution: Update go-version in workflow files to match go.mod

### Install script download fails
Solution: Ensure script is in main branch:
```bash
git checkout main
git push origin main
```

## Security Considerations

### Secrets Management
- API keys go in `~/.kai/credentials.json` (user home)
- Never commit `.env` or config files with secrets
- GitHub Actions uses automatic GITHUB_TOKEN (safe)

### Binary Integrity
- All binaries are checksummed (SHA256)
- Users can verify with `sha256sum -c`
- Consider code signing for production (macOS/Windows)

### Source Code Security
- No credentials in repository
- No hardcoded API endpoints
- Install script uses HTTPS only
- No privilege escalation (requires sudo for system install)

## Automation Setup (Optional)

To automatically build on git tags:

Edit `.github/workflows/build-release.yml` and change:

```yaml
on:
  workflow_dispatch:
    inputs:
      version: ...
```

To:

```yaml
on:
  push:
    tags:
      - "v*"
  workflow_dispatch:
    inputs:
      version: ...
```

Then releases auto-trigger on `git tag v1.0.0 && git push --tags`

## Next Steps

1. ✅ Create both GitHub repositories
2. ✅ Push source to tinhvqbk/kai-code
3. ✅ Configure GitHub Actions permissions
4. ✅ Run first test build
5. ✅ Verify release appears in tinhvqbk/kai
6. ✅ Test install.sh manually
7. ✅ Announce to users

## Release Cadence

Recommended: **Monthly security updates + ad-hoc feature releases**

- Patch releases (v1.0.1): When bugs are fixed
- Minor releases (v1.1.0): When features are added
- Major releases (v2.0.0): When breaking changes occur

## Reference

- GitHub Actions: https://docs.github.com/en/actions
- Go build: https://golang.org/cmd/go/#hdr-Compile
- Homebrew: https://docs.brew.sh/Formula-Cookbook
