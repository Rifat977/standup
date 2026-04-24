package formatter

import (
	"fmt"
	"strings"

	"github.com/rifat977/standup/internal/ai"
	gitscan "github.com/rifat977/standup/internal/git"
)

type Format int

const (
	Markdown Format = iota
	Plain
	Slack
)

// RenderSummary formats an AI summary string for the chosen output medium.
func RenderSummary(summary string, f Format) string {
	switch f {
	case Slack:
		// Slack mrkdwn: bold uses *single*, not **double**.
		s := strings.ReplaceAll(summary, "**", "*")
		return s + "\n"
	case Plain:
		// Strip markdown bold markers.
		s := strings.ReplaceAll(summary, "**", "")
		return s + "\n"
	default:
		return summary + "\n"
	}
}

// RenderRaw produces a deterministic standup-style block from the raw inputs
// (used by `standup show --no-ai`).
func RenderRaw(d ai.Data, f Format) string {
	var b strings.Builder
	groups, order := gitscan.GroupByRepo(d.Commits)

	switch f {
	case Slack:
		fmt.Fprintln(&b, "*Commits & PRs*")
	case Plain:
		fmt.Fprintln(&b, "Commits & PRs")
		fmt.Fprintln(&b, "-------------")
	default:
		fmt.Fprintln(&b, "## Commits & PRs")
	}

	if len(d.Commits) == 0 && len(d.PRs) == 0 {
		fmt.Fprintln(&b, "(no activity in window)")
	}

	for _, repo := range order {
		fmt.Fprintf(&b, "\n%s\n", repoHeader(repo, f))
		for _, c := range groups[repo] {
			fmt.Fprintf(&b, "  %s  %s\n", c.Hash, c.Subject)
		}
	}

	if len(d.PRs) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, sectionHeader("Pull Requests", f))
		for _, p := range d.PRs {
			fmt.Fprintf(&b, "  #%d  %s  [%s/%s/CI %s]  %s\n",
				p.Number, p.Title, strings.ToUpper(p.State),
				strings.ToUpper(p.Review), p.CI, p.AgeString())
		}
	}

	if strings.TrimSpace(d.Today) != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, sectionHeader("Today", f))
		fmt.Fprintln(&b, "  "+d.Today)
	}
	if strings.TrimSpace(d.Blocker) != "" {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, sectionHeader("Blockers", f))
		fmt.Fprintln(&b, "  "+d.Blocker)
	}
	return b.String()
}

func sectionHeader(title string, f Format) string {
	switch f {
	case Slack:
		return "*" + title + "*"
	case Plain:
		return title
	default:
		return "## " + title
	}
}

func repoHeader(repo string, f Format) string {
	switch f {
	case Slack:
		return "_" + repo + "_"
	case Plain:
		return "── " + repo + " ──"
	default:
		return "### " + repo
	}
}
