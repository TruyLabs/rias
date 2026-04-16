package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	kai "github.com/TruyLabs/rias"
	"github.com/TruyLabs/rias/internal/config"
)

const (
	updateCheckURL      = "https://api.github.com/repos/norenis/kai/releases/latest"
	updateCheckInterval = 24 * time.Hour
)

// checkForUpdate runs a background version check and prints a notice if a
// newer release is available. It is rate-limited to once per day via a
// timestamp file in ~/.kai/.update_check.
//
// Call as `go checkForUpdate()` — it is intentionally non-blocking.
func checkForUpdate() {
	if kai.Version == "dev" {
		return // don't nag during local development
	}

	if !shouldCheck() {
		return
	}

	latest, err := fetchLatestVersion()
	if err != nil || latest == "" {
		return // silent on network errors
	}

	writeCheckTimestamp()

	if isNewer(latest, kai.Version) {
		fmt.Fprintf(os.Stderr, "\n\033[33m» New version available: %s → %s\033[0m\n", kai.Version, latest)
		fmt.Fprintf(os.Stderr, "  Update with: \033[1mbrew upgrade %s\033[0m\n\n", config.DefaultAgentName)
	}
}

// shouldCheck returns true if 24 hours have passed since the last check.
func shouldCheck() bool {
	ts, err := readCheckTimestamp()
	if err != nil {
		return true
	}
	return time.Since(ts) >= updateCheckInterval
}

func timestampFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "."+config.DefaultAgentName, ".update_check")
}

func readCheckTimestamp() (time.Time, error) {
	f := timestampFile()
	if f == "" {
		return time.Time{}, fmt.Errorf("no home dir")
	}
	data, err := os.ReadFile(f)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
}

func writeCheckTimestamp() {
	f := timestampFile()
	if f == "" {
		return
	}
	_ = os.WriteFile(f, []byte(time.Now().UTC().Format(time.RFC3339)), 0644)
}

func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(updateCheckURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.TagName, nil
}

// isNewer returns true if candidate is a higher semver than current.
// Both strings may have a leading "v".
func isNewer(candidate, current string) bool {
	c := parseSemver(strings.TrimPrefix(candidate, "v"))
	cur := parseSemver(strings.TrimPrefix(current, "v"))
	for i := range c {
		if i >= len(cur) {
			return true
		}
		if c[i] > cur[i] {
			return true
		}
		if c[i] < cur[i] {
			return false
		}
	}
	return false
}

func parseSemver(s string) []int {
	parts := strings.SplitN(s, ".", 3)
	out := make([]int, 3)
	for i, p := range parts {
		if i >= 3 {
			break
		}
		fmt.Sscanf(p, "%d", &out[i])
	}
	return out
}
