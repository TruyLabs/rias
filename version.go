package kai

// Version information. Overridden at build time via ldflags:
//
//	go build -ldflags "-X github.com/tinhvqbk/kai.Version=v1.0.0 -X github.com/tinhvqbk/kai.Commit=$(git rev-parse --short HEAD) -X github.com/tinhvqbk/kai.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)
