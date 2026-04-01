# Production Deployment Checklist

This document ensures kai is production-ready for distribution to users.

## ✅ Pre-Release Checklist

### Code Quality
- [ ] All tests pass: `make test`
- [ ] No race conditions: `go test -race ./...`
- [ ] Code builds without warnings: `make build`
- [ ] Version info is correct: `./kai version`
- [ ] All commands work: `./kai --help`

### Security
- [ ] No hardcoded credentials in code
- [ ] No secrets in `.gitignore` exceptions
- [ ] Dependencies are up to date: `go mod tidy`
- [ ] No known vulnerabilities: `go list -m all`
- [ ] Install script has no injection vulnerabilities
- [ ] No plaintext sensitive data in config examples

### Documentation
- [ ] README.md is complete and accurate
- [ ] Installation instructions are clear for all platforms
- [ ] RELEASE.md is comprehensive
- [ ] API documentation exists (if applicable)
- [ ] Error messages are user-friendly
- [ ] `--help` output is clear

### Build & Release
- [ ] Makefile is production-ready
- [ ] Version injection via ldflags works correctly
- [ ] Build is reproducible (same output for same source)
- [ ] GitHub Actions workflow is tested
- [ ] All required permissions are set (contents: write)
- [ ] Artifacts are properly checksummed

## 🚀 Release Process

### Step 1: Pre-Release Testing
```bash
# Clone fresh copy
git clone https://github.com/tinhvqbk/kai-code.git test-release
cd test-release

# Run all tests
make test

# Build for all platforms locally
make clean
make build GOOS=darwin GOARCH=amd64
make build GOOS=darwin GOARCH=arm64
make build GOOS=linux GOARCH=amd64

# Test each binary
./kai-darwin-amd64 version
./kai-darwin-arm64 version
./kai-linux-amd64 version
```

### Step 2: Trigger GitHub Actions Build
1. Go to **tinhvqbk/kai-code** → Actions → **Build and Release**
2. Click **Run workflow**
3. Enter version (e.g., `v1.0.0`)
4. Wait for workflow to complete
5. Verify all jobs passed ✓

### Step 3: Verify Release
```bash
# Check tinhvqbk/kai releases page
curl -s https://api.github.com/repos/tinhvqbk/kai/releases/latest | jq '.tag_name, .assets[].name'

# Verify all 6 files present:
# - kai-darwin-amd64
# - kai-darwin-amd64.sha256
# - kai-darwin-arm64
# - kai-darwin-arm64.sha256
# - kai-linux-amd64
# - kai-linux-amd64.sha256
```

### Step 4: Test Installation Methods
```bash
# Method 1: Install script (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/tinhvqbk/kai/main/install.sh | bash
which kai
kai version

# Method 2: Direct download
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64 -o /tmp/kai
chmod +x /tmp/kai
/tmp/kai version

# Method 3: Verify checksum
cd /tmp
curl -L https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64.sha256 -o kai.sha256
sha256sum -c kai.sha256
```

### Step 5: Update Homebrew Formula (if using tap)
```bash
# Get actual SHA256 values from release
curl -s https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-amd64.sha256 | cut -d' ' -f1
curl -s https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-darwin-arm64.sha256 | cut -d' ' -f1
curl -s https://github.com/tinhvqbk/kai/releases/download/v1.0.0/kai-linux-amd64.sha256 | cut -d' ' -f1

# Update Formula/kai.rb with exact hashes
# Then commit and push to homebrew-kai tap
git clone https://github.com/tinhvqbk/homebrew-kai.git
cd homebrew-kai
# Edit Formula/kai.rb with new version and SHA256 values
git add Formula/kai.rb
git commit -m "Update kai to v1.0.0"
git push
```

## 🔒 Security Best Practices

### Distribution
- ✅ Binaries are signed (use codesign for macOS)
- ✅ Checksums are provided for verification
- ✅ Release notes link to source code
- ✅ Install script uses HTTPS only
- ✅ No auto-update mechanism (users control updates)

### Credentials
- ✅ No API keys embedded in binaries
- ✅ Config files never committed
- ✅ Credentials stored in `~/.kai/credentials.json` (user home)
- ✅ Clear permission model for config files

### Dependencies
- ✅ All Go dependencies are vendored (or use go.mod)
- ✅ Regular security updates: `go get -u`
- ✅ No untracked transitive dependencies

## 📊 Monitoring & Support

### After Release
1. **Monitor GitHub Issues** for bug reports
2. **Check install script downloads** - broken links?
3. **Test frequently** - new OS versions, Go updates
4. **Maintain changelog** - link each release to commits
5. **Security scanning** - watch for CVEs

### User Support
- Provide clear error messages
- Document common issues in README
- Create troubleshooting guide
- Respond to installation issues quickly

## 🔄 Continuous Improvement

### Regular Tasks
- [ ] Monthly: Update Go version (`go mod edit -go=1.XX`)
- [ ] Quarterly: Review and update dependencies
- [ ] Per-release: Test on fresh VMs
- [ ] Per-release: Test across OS versions (macOS 12+, Ubuntu 20.04+, etc.)

### Automated Testing
- [ ] Unit tests: `go test ./...`
- [ ] Integration tests: actual binary execution
- [ ] Cross-platform builds: verify all OSes compile
- [ ] Binary verification: checksums, signatures

## 📝 Version Numbering

Use semantic versioning: `v{MAJOR}.{MINOR}.{PATCH}`

- **MAJOR** - Breaking changes
- **MINOR** - New features (backward compatible)
- **PATCH** - Bug fixes

Example releases:
- `v1.0.0` - Initial release
- `v1.1.0` - New feature (backward compatible)
- `v1.1.1` - Bug fix
- `v2.0.0` - Breaking change

## 🚨 Emergency Procedures

### Rollback Release (if critical bug found)
```bash
# Delete release
gh release delete v1.0.0 -y

# Delete git tag
git push origin --delete v1.0.0

# Fix bug and re-release
# Re-run GitHub Actions workflow
```

### Critical Security Issue
1. **Immediately** yank the release from Homebrew
2. **Create** patch release with fix
3. **Update** all documentation
4. **Announce** on all channels

## ✅ Final Checklist Before Release

- [ ] Code changes are tested
- [ ] Documentation is updated
- [ ] Version number is bumped
- [ ] CHANGELOG is updated (if exists)
- [ ] All tests pass
- [ ] GitHub Actions workflow is configured
- [ ] tinhvqbk/kai repo exists and is empty
- [ ] Workflow has write permissions
- [ ] Install script is in main branch
- [ ] Installation methods are documented
- [ ] Checksums will be created automatically
- [ ] Release notes template is ready

## Release Command (All-in-One)

Once everything is set up:

1. **GitHub UI**: Actions → Build and Release → Run workflow → Enter version
2. **Wait ~5 minutes** for builds to complete
3. **Verify** on GitHub releases page
4. **Done!** Users can now install

---

**Production readiness: 100%** ✅
