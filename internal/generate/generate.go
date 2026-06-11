// Package generate turns an Analysis into Conventional Commit messages and
// pull request descriptions.
package generate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/djaferiurim/gitmsg/internal/analyze"
	"github.com/djaferiurim/gitmsg/internal/git"
)

// CommitMessage builds a Conventional Commit message from an Analysis.
//
// The format is:
//
//	type(scope): summary
//
//	- changed file one
//	- changed file two
func CommitMessage(a analyze.Analysis, body bool) string {
	var header strings.Builder
	header.WriteString(a.Type)
	if a.Scope != "" {
		header.WriteString("(" + a.Scope + ")")
	}
	if a.Breaking {
		header.WriteString("!")
	}
	header.WriteString(": ")
	header.WriteString(a.Summary)

	msg := header.String()
	if !body || len(a.Files) <= 1 {
		return msg
	}

	lines := make([]string, 0, len(a.Files))
	for _, f := range a.Files {
		lines = append(lines, "- "+describeChange(f))
	}
	return msg + "\n\n" + strings.Join(lines, "\n")
}

func describeChange(f git.FileChange) string {
	p := strings.ReplaceAll(f.Path, "\\", "/")
	switch f.Status {
	case "A":
		return "add " + p
	case "D":
		return "remove " + p
	case "R":
		return "rename " + strings.ReplaceAll(f.OldPath, "\\", "/") + " -> " + p
	default:
		return "update " + p
	}
}

var ccPrefix = regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)(\([^)]*\))?!?:\s`)

// PullRequest builds a PR title and body from the commits on a branch.
func PullRequest(commits []git.Commit, branch string) (title, body string) {
	if len(commits) == 0 {
		return "", ""
	}

	// The title comes from the oldest commit (usually the first meaningful one)
	// when there are several, otherwise the single commit subject.
	title = commits[len(commits)-1].Subject
	if len(commits) == 1 {
		title = commits[0].Subject
	}

	groups := map[string][]string{}
	order := []string{}
	for _, c := range commits {
		section := sectionFor(c.Subject)
		clean := ccPrefix.ReplaceAllString(c.Subject, "")
		if _, ok := groups[section]; !ok {
			order = append(order, section)
		}
		groups[section] = append(groups[section], clean)
	}
	sort.Slice(order, func(i, j int) bool {
		return sectionRank(order[i]) < sectionRank(order[j])
	})

	var b strings.Builder
	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("This pull request introduces %d change(s) on `%s`.\n\n", len(commits), branch))
	b.WriteString("## Changes\n\n")
	for _, section := range order {
		b.WriteString("### " + section + "\n\n")
		for _, item := range groups[section] {
			b.WriteString("- " + capitalize(item) + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Checklist\n\n")
	b.WriteString("- [ ] Tests added or updated\n")
	b.WriteString("- [ ] Documentation updated\n")
	return capitalize(strings.TrimSpace(ccPrefix.ReplaceAllString(title, ""))), strings.TrimSpace(b.String())
}

func sectionFor(subject string) string {
	m := ccPrefix.FindStringSubmatch(subject)
	if len(m) == 0 {
		return "Other"
	}
	switch m[1] {
	case "feat":
		return "Features"
	case "fix":
		return "Bug Fixes"
	case "docs":
		return "Documentation"
	case "perf":
		return "Performance"
	case "refactor":
		return "Refactoring"
	case "test":
		return "Tests"
	case "build", "ci", "chore", "style":
		return "Maintenance"
	default:
		return "Other"
	}
}

func sectionRank(s string) int {
	switch s {
	case "Features":
		return 0
	case "Bug Fixes":
		return 1
	case "Performance":
		return 2
	case "Refactoring":
		return 3
	case "Documentation":
		return 4
	case "Tests":
		return 5
	case "Maintenance":
		return 6
	default:
		return 7
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
