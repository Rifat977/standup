package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	gh "github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"

	"github.com/rifat977/standup/internal/config"
	"github.com/rifat977/standup/internal/logx"
)

// PR is the minimal PR record we display and pass to the AI.
type PR struct {
	Repo      string
	Number    int
	Title     string
	Author    string
	State     string // open / merged / closed
	Review    string // approved / changes_requested / pending
	CI        string // pass / fail / pending
	UpdatedAt time.Time
}

// Collect fetches PRs updated within cfg.Since across all configured repos.
func Collect(ctx context.Context, cfg *config.Config) ([]PR, error) {
	if cfg.GitHub.Token == "" {
		logx.Warn("github: token not set — skipping PR fetch (set github.token or GITHUB_TOKEN)")
		return nil, nil
	}
	if len(cfg.GitHub.Repos) == 0 {
		logx.Warn("github: no repos configured — skipping PR fetch")
		return nil, nil
	}
	since, err := time.ParseDuration(cfg.Since)
	if err != nil {
		logx.Warn("github: invalid since=%q (%v); falling back to 12h", cfg.Since, err)
		since = 12 * time.Hour
	}
	cutoff := time.Now().Add(-since)

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: cfg.GitHub.Token})
	tc := oauth2.NewClient(ctx, ts)
	client := gh.NewClient(tc)

	var out []PR
	for _, slug := range cfg.GitHub.Repos {
		owner, repo, ok := splitSlug(slug)
		if !ok {
			logx.Warn("github: invalid repo slug %q (expected owner/repo)", slug)
			continue
		}
		opts := &gh.PullRequestListOptions{
			State:       "all",
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: gh.ListOptions{PerPage: 50},
		}
		prs, resp, err := client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			status := ""
			if resp != nil {
				status = resp.Status
			}
			logx.Warn("github: list PRs %s/%s failed (%s): %v", owner, repo, status, err)
			continue
		}
		matched := 0
		for _, p := range prs {
			updated := p.GetUpdatedAt().Time
			if updated.Before(cutoff) {
				break // sorted desc
			}
			out = append(out, PR{
				Repo:      repo,
				Number:    p.GetNumber(),
				Title:     p.GetTitle(),
				Author:    p.GetUser().GetLogin(),
				State:     prState(p),
				Review:    reviewState(ctx, client, owner, repo, p.GetNumber()),
				CI:        ciState(ctx, client, owner, repo, p.GetHead().GetSHA()),
				UpdatedAt: updated,
			})
			matched++
		}
		logx.Info("github: %s/%s — %d PR(s) within last %s", owner, repo, matched, since)
	}
	logx.Info("github: collected %d PR(s) total", len(out))
	return out, nil
}

func splitSlug(s string) (string, string, bool) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func prState(p *gh.PullRequest) string {
	if p.GetMerged() {
		return "merged"
	}
	return p.GetState()
}

func reviewState(ctx context.Context, c *gh.Client, owner, repo string, num int) string {
	revs, _, err := c.PullRequests.ListReviews(ctx, owner, repo, num, &gh.ListOptions{PerPage: 50})
	if err != nil {
		logx.Debug("github: list reviews %s/%s#%d failed: %v", owner, repo, num, err)
		return "pending"
	}
	if len(revs) == 0 {
		return "pending"
	}
	// last non-comment review wins
	for i := len(revs) - 1; i >= 0; i-- {
		state := revs[i].GetState()
		switch state {
		case "APPROVED":
			return "approved"
		case "CHANGES_REQUESTED":
			return "changes_requested"
		}
	}
	return "pending"
}

func ciState(ctx context.Context, c *gh.Client, owner, repo, sha string) string {
	if sha == "" {
		return "pending"
	}
	checks, _, err := c.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, &gh.ListCheckRunsOptions{})
	if err != nil {
		logx.Debug("github: list checks %s/%s sha=%s failed: %v", owner, repo, sha, err)
		return "pending"
	}
	if checks.GetTotal() == 0 {
		return "pending"
	}
	worst := "pass"
	for _, run := range checks.CheckRuns {
		if run.GetStatus() != "completed" {
			worst = "pending"
			continue
		}
		if run.GetConclusion() == "failure" || run.GetConclusion() == "timed_out" {
			return "fail"
		}
	}
	return worst
}

// AgeString returns a short relative age like "2h ago".
func (p PR) AgeString() string {
	d := time.Since(p.UpdatedAt)
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
