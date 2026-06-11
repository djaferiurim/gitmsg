package generate

import (
	"strings"
	"testing"

	"github.com/djaferiurim/gitmsg/internal/analyze"
	"github.com/djaferiurim/gitmsg/internal/git"
)

func TestCommitMessageHeader(t *testing.T) {
	a := analyze.Analysis{Type: "feat", Scope: "git", Summary: "add staging helpers"}
	got := CommitMessage(a, true)
	want := "feat(git): add staging helpers"
	if got != want {
		t.Fatalf("CommitMessage = %q, want %q", got, want)
	}
}

func TestCommitMessageNoScope(t *testing.T) {
	a := analyze.Analysis{Type: "fix", Summary: "handle empty diff"}
	if got := CommitMessage(a, true); got != "fix: handle empty diff" {
		t.Fatalf("CommitMessage = %q", got)
	}
}

func TestCommitMessageBreaking(t *testing.T) {
	a := analyze.Analysis{Type: "feat", Scope: "api", Summary: "drop v1 routes", Breaking: true}
	if got := CommitMessage(a, false); got != "feat(api)!: drop v1 routes" {
		t.Fatalf("CommitMessage = %q", got)
	}
}

func TestCommitMessageBody(t *testing.T) {
	a := analyze.Analysis{
		Type:    "feat",
		Scope:   "git",
		Summary: "update git",
		Files: []git.FileChange{
			{Status: "A", Path: "internal/git/a.go"},
			{Status: "M", Path: "internal/git/b.go"},
		},
	}
	got := CommitMessage(a, true)
	if !strings.Contains(got, "- add internal/git/a.go") {
		t.Fatalf("body missing add line:\n%s", got)
	}
	if !strings.Contains(got, "- update internal/git/b.go") {
		t.Fatalf("body missing update line:\n%s", got)
	}
}

func TestPullRequest(t *testing.T) {
	commits := []git.Commit{
		{Subject: "feat(git): add staging"},
		{Subject: "fix(cli): handle empty diff"},
		{Subject: "docs: update readme"},
	}
	title, body := PullRequest(commits, "feature/x")
	if title == "" {
		t.Fatal("expected non-empty title")
	}
	for _, section := range []string{"### Features", "### Bug Fixes", "### Documentation"} {
		if !strings.Contains(body, section) {
			t.Fatalf("body missing %q:\n%s", section, body)
		}
	}
	// Features should appear before Bug Fixes.
	if strings.Index(body, "### Features") > strings.Index(body, "### Bug Fixes") {
		t.Fatal("sections out of order")
	}
}
