package module

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/norenis/kai/internal/brain"
)

const githubAPIBase = "https://api.github.com"

// gitHubPRsModule fetches pull requests from one or more GitHub repositories.
type gitHubPRsModule struct {
	token string
	repos []string
	state string
	limit int
}

// NewGitHubPRsModule creates a github_prs module from a raw config map.
//
// Required config keys:
//   - token: GitHub personal access token
//   - repos: list of "owner/repo" strings
//
// Optional config keys:
//   - state: "open" | "closed" | "all" (default: "open")
//   - limit: max PRs to fetch per repo (default: 20)
func NewGitHubPRsModule(cfg map[string]interface{}) (Module, error) {
	token := cfgString(cfg, "token", "")
	if token == "" {
		return nil, fmt.Errorf("github_prs: 'token' is required")
	}
	repos := cfgStrings(cfg, "repos")
	if len(repos) == 0 {
		return nil, fmt.Errorf("github_prs: 'repos' is required (e.g. [\"owner/repo\"])")
	}
	return &gitHubPRsModule{
		token: token,
		repos: repos,
		state: cfgString(cfg, "state", "open"),
		limit: cfgInt(cfg, "limit", 20),
	}, nil
}

func (m *gitHubPRsModule) Name() string        { return "github_prs" }
func (m *gitHubPRsModule) Description() string { return "Read GitHub pull requests into the brain" }

type ghPR struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	User    struct {
		Login string `json:"login"`
	} `json:"user"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	MergedAt  *string `json:"merged_at"`
	CreatedAt string  `json:"created_at"`
}

func (m *gitHubPRsModule) Fetch(ctx context.Context) ([]brain.Learning, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	var learnings []brain.Learning

	for _, repo := range m.repos {
		prs, err := m.fetchPRs(ctx, client, repo)
		if err != nil {
			return nil, fmt.Errorf("github_prs: %s: %w", repo, err)
		}
		for _, pr := range prs {
			learnings = append(learnings, m.prToLearning(pr, repo))
		}
	}

	return learnings, nil
}

func (m *gitHubPRsModule) fetchPRs(ctx context.Context, client *http.Client, repo string) ([]ghPR, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls?state=%s&per_page=%d", githubAPIBase, repo, m.state, m.limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, string(body))
	}

	var prs []ghPR
	if err := json.NewDecoder(resp.Body).Decode(&prs); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return prs, nil
}

const prBodyMaxLen = 400

func (m *gitHubPRsModule) prToLearning(pr ghPR, repo string) brain.Learning {
	repoSlug := strings.NewReplacer("/", "-", "_", "-").Replace(repo)

	status := pr.State
	if pr.MergedAt != nil {
		status = "merged"
	}

	var labelNames []string
	for _, l := range pr.Labels {
		labelNames = append(labelNames, l.Name)
	}

	// Compact single-line header: minimal markdown, no decorative bold labels.
	// Saves ~50 tokens per entry vs verbose markdown headings.
	meta := fmt.Sprintf("PR #%d [%s] %s @%s", pr.Number, status, repo, pr.User.Login)
	if len(labelNames) > 0 {
		meta += " labels:" + strings.Join(labelNames, ",")
	}

	body := strings.TrimSpace(pr.Body)
	if body == "" {
		body = "no description"
	} else if len(body) > prBodyMaxLen {
		body = body[:prBodyMaxLen] + "…"
	}

	content := fmt.Sprintf("%s\n%s\n%s\n", meta, pr.HTMLURL, body)

	tags := append([]string{"github", "pull-request", repoSlug}, labelNames...)

	return brain.Learning{
		Category:   "knowledge",
		Topic:      fmt.Sprintf("github-%s-pr-%d", repoSlug, pr.Number),
		Tags:       tags,
		Content:    content,
		Confidence: "high",
		Action:     "create",
		Source:     "module:github_prs",
	}
}
