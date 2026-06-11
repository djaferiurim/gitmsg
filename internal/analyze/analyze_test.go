package analyze

import (
	"testing"

	"github.com/djaferiurim/gitmsg/internal/git"
)

func TestInferType(t *testing.T) {
	cases := []struct {
		name  string
		files []git.FileChange
		want  string
	}{
		{"docs only", []git.FileChange{{Status: "M", Path: "README.md"}}, "docs"},
		{"docs dir", []git.FileChange{{Status: "A", Path: "docs/guide.txt"}}, "docs"},
		{"tests", []git.FileChange{{Status: "A", Path: "internal/git/git_test.go"}}, "test"},
		{"ci", []git.FileChange{{Status: "M", Path: ".github/workflows/ci.yml"}}, "ci"},
		{"build", []git.FileChange{{Status: "M", Path: "go.mod"}}, "build"},
		{"new feature", []git.FileChange{{Status: "A", Path: "internal/feature/x.go"}}, "feat"},
		{"modification", []git.FileChange{{Status: "M", Path: "internal/feature/x.go"}}, "fix"},
		{"deletion only", []git.FileChange{{Status: "D", Path: "internal/old/x.go"}}, "chore"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferType(tc.files); got != tc.want {
				t.Fatalf("inferType = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInferScope(t *testing.T) {
	cases := []struct {
		name  string
		files []git.FileChange
		want  string
	}{
		{"single nested", []git.FileChange{{Status: "M", Path: "internal/git/git.go"}}, "git"},
		{"top-level file", []git.FileChange{{Status: "M", Path: "main.go"}}, "main"},
		{"common dir", []git.FileChange{
			{Status: "M", Path: "internal/git/a.go"},
			{Status: "M", Path: "internal/git/b.go"},
		}, "git"},
		{"no common dir", []git.FileChange{
			{Status: "M", Path: "internal/git/a.go"},
			{Status: "M", Path: "cmd/main.go"},
		}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferScope(tc.files); got != tc.want {
				t.Fatalf("inferScope = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestInferSummary(t *testing.T) {
	cases := []struct {
		name  string
		files []git.FileChange
		scope string
		want  string
	}{
		{"add single", []git.FileChange{{Status: "A", Path: "internal/git/git.go"}}, "git", "add git.go"},
		{"remove single", []git.FileChange{{Status: "D", Path: "old.go"}}, "old", "remove old.go"},
		{"update single", []git.FileChange{{Status: "M", Path: "main.go"}}, "main", "update main.go"},
		{"multi update", []git.FileChange{
			{Status: "M", Path: "internal/git/a.go"},
			{Status: "M", Path: "internal/git/b.go"},
		}, "git", "update git"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferSummary(tc.files, tc.scope); got != tc.want {
				t.Fatalf("inferSummary = %q, want %q", got, tc.want)
			}
		})
	}
}
