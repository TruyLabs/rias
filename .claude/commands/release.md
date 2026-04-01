Prepare a kai release.

Version: `$ARGUMENTS` (e.g. v1.2.3)

Pre-release checklist:
1. Run all tests: `make test`
2. Build and verify binary: `make build && ./kai version`
3. Check git status is clean: `git status`
4. Review changes since last tag: `git log $(git describe --tags --abbrev=0)..HEAD --oneline`
5. Verify the GitHub Actions workflow: `cat .github/workflows/build-release.yml`

If everything looks good, show me the git commands needed to tag and push — but DO NOT run them without explicit confirmation:
```bash
git tag -a $ARGUMENTS -m "Release $ARGUMENTS"
git push origin $ARGUMENTS
```

The GitHub Action will automatically build binaries for all platforms and create the GitHub release.
