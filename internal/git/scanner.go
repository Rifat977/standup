package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rifat977/standup/internal/config"
	"github.com/rifat977/standup/internal/logx"
)

// Commit is a single git commit parsed from `git log`.
type Commit struct {
	Repo    string
	Hash    string
	Subject string
	Author  string
	Date    string
}

// Collect walks all configured scan_dirs one level deep, runs `git log` in each
// repo, and returns flattened commits. Errors in individual repos are skipped.
func Collect(cfg *config.Config) ([]Commit, error) {
	var out []Commit
	if len(cfg.ScanDirs) == 0 {
		logx.Warn("git: no scan_dirs configured")
		return nil, nil
	}
	for _, dir := range cfg.ScanDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			logx.Warn("git: cannot read scan_dir %q: %v", dir, err)
			continue
		}
		repoCount := 0
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			repoPath := filepath.Join(dir, e.Name())
			if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
				continue
			}
			repoCount++
			commits, err := repoLog(repoPath, cfg)
			if err != nil {
				logx.Warn("git: log failed for %s: %v", repoPath, err)
				continue
			}
			logx.Debug("git: %s yielded %d commit(s)", filepath.Base(repoPath), len(commits))
			out = append(out, commits...)
		}
		logx.Info("git: scanned %s (%d repos)", dir, repoCount)
	}
	logx.Info("git: collected %d commit(s) total", len(out))
	return out, nil
}

func repoLog(repoPath string, cfg *config.Config) ([]Commit, error) {
	args := []string{
		"log",
		"--format=%H|%s|%an|%ai",
		"--since=" + cfg.Since,
	}
	if cfg.Author != "" {
		args = append(args, "--author="+cfg.Author)
	}
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	logx.Debug("git: %s $ git %s", repoPath, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%v: %s", err, msg)
		}
		return nil, err
	}
	repoName := filepath.Base(repoPath)
	var commits []Commit
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		commits = append(commits, Commit{
			Repo:    repoName,
			Hash:    shortHash(parts[0]),
			Subject: parts[1],
			Author:  parts[2],
			Date:    parts[3],
		})
	}
	return commits, nil
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// GroupByRepo returns commits grouped by repo name, preserving insertion order.
func GroupByRepo(commits []Commit) (map[string][]Commit, []string) {
	groups := map[string][]Commit{}
	var order []string
	for _, c := range commits {
		if _, ok := groups[c.Repo]; !ok {
			order = append(order, c.Repo)
		}
		groups[c.Repo] = append(groups[c.Repo], c)
	}
	return groups, order
}
