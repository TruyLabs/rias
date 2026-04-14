package module

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/norenis/kai/internal/brain"
)

const githubAPIBase = "https://api.github.com"

// gitHubPRsModule fetches pull requests from one or more GitHub repositories.
type gitHubPRsModule struct {
	token string // empty = use gh CLI
	repos []string
	state string
	limit int
}

// NewGitHubPRsModule creates a github_prs module from a raw config map.
//
// Required config keys:
//   - repos: list of "owner/repo" strings
//
// Optional config keys:
//   - token: GitHub personal access token (if empty, uses `gh` CLI)
//   - state: "open" | "closed" | "all" (default: "open")
//   - limit: max PRs to fetch per repo (default: 20)
func NewGitHubPRsModule(cfg map[string]interface{}) (Module, error) {
	token := cfgString(cfg, "token", "")
	if token == "" {
		// No token — check if gh CLI is available and authenticated.
		if _, err := exec.LookPath("gh"); err != nil {
			return nil, fmt.Errorf("github_prs: 'token' not set and 'gh' CLI not found — provide a token or install gh (https://cli.github.com)")
		}
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
	var learnings []brain.Learning

	for _, repo := range m.repos {
		var prs []ghPR
		var err error
		if m.token != "" {
			prs, err = m.fetchPRsAPI(ctx, repo)
		} else {
			prs, err = m.fetchPRsGH(ctx, repo)
		}
		if err != nil {
			return nil, fmt.Errorf("github_prs: %s: %w", repo, err)
		}
		for _, pr := range prs {
			learnings = append(learnings, m.prToLearning(pr, repo))
		}
	}

	return learnings, nil
}

// fetchPRsAPI fetches PRs using the GitHub REST API with a token.
func (m *gitHubPRsModule) fetchPRsAPI(ctx context.Context, repo string) ([]ghPR, error) {
	client := &http.Client{Timeout: 30 * time.Second}
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

// fetchPRsGH fetches PRs using the gh CLI (no token needed).
func (m *gitHubPRsModule) fetchPRsGH(ctx context.Context, repo string) ([]ghPR, error) {
	state := m.state
	if state == "" {
		state = "open"
	}

	args := []string{"pr", "list",
		"--repo", repo,
		"--state", state,
		"--limit", fmt.Sprintf("%d", m.limit),
		"--json", "number,title,body,state,url,author,labels,mergedAt,createdAt",
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh cli: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh cli: %w", err)
	}

	// gh outputs a different JSON shape — map it to ghPR.
	var ghResults []struct {
		Number   int    `json:"number"`
		Title    string `json:"title"`
		Body     string `json:"body"`
		State    string `json:"state"`
		URL      string `json:"url"`
		Author   struct {
			Login string `json:"login"`
		} `json:"author"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
		MergedAt  *string `json:"mergedAt"`
		CreatedAt string  `json:"createdAt"`
	}
	if err := json.Unmarshal(out, &ghResults); err != nil {
		return nil, fmt.Errorf("decode gh output: %w", err)
	}

	prs := make([]ghPR, len(ghResults))
	for i, r := range ghResults {
		prs[i] = ghPR{
			Number:    r.Number,
			Title:     r.Title,
			Body:      r.Body,
			State:     strings.ToLower(r.State),
			HTMLURL:   r.URL,
			MergedAt:  r.MergedAt,
			CreatedAt: r.CreatedAt,
		}
		prs[i].User.Login = r.Author.Login
		prs[i].Labels = make([]struct {
			Name string `json:"name"`
		}, len(r.Labels))
		copy(prs[i].Labels, r.Labels)
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
