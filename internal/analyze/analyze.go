// Package analyze inspects staged git changes and infers a Conventional
// Commit type, scope, and a short imperative summary using local heuristics.
package analyze

import (
	"path"
	"sort"
	"strings"

	"github.com/djaferiurim/gitmsg/internal/git"
)

// Analysis is the structured result of inspecting staged changes.
type Analysis struct {
	Type       string
	Scope      string
	Summary    string
	Breaking   bool
	Files      []git.FileChange
	Insertions int
	Deletions  int
}

// Changes computes an Analysis from staged files and diff statistics.
func Changes(files []git.FileChange, insertions, deletions int) Analysis {
	a := Analysis{
		Files:      files,
		Insertions: insertions,
		Deletions:  deletions,
	}
	a.Type = inferType(files)
	a.Scope = inferScope(files)
	a.Summary = inferSummary(files, a.Scope)
	return a
}

func inferType(files []git.FileChange) string {
	if len(files) == 0 {
		return "chore"
	}

	allMatch := func(pred func(string) bool) bool {
		for _, f := range files {
			if !pred(f.Path) {
				return false
			}
		}
		return true
	}

	switch {
	case allMatch(isCI):
		return "ci"
	case allMatch(isDocs):
		return "docs"
	case allMatch(isTest):
		return "test"
	case allMatch(isBuild):
		return "build"
	}

	// New files with no modifications usually means a new feature.
	onlyAdded := true
	anyModified := false
	for _, f := range files {
		if f.Status != "A" {
			onlyAdded = false
		}
		if f.Status == "M" {
			anyModified = true
		}
	}
	if onlyAdded {
		return "feat"
	}
	if anyModified {
		return "fix"
	}
	return "chore"
}

func inferScope(files []git.FileChange) string {
	if len(files) == 1 {
		return scopeFromPath(files[0].Path)
	}
	// Find the longest common directory prefix across all files.
	dirs := make([][]string, 0, len(files))
	for _, f := range files {
		dir := path.Dir(toSlash(f.Path))
		if dir == "." || dir == "/" {
			return ""
		}
		dirs = append(dirs, strings.Split(dir, "/"))
	}
	common := dirs[0]
	for _, d := range dirs[1:] {
		common = commonPrefix(common, d)
		if len(common) == 0 {
			return ""
		}
	}
	return cleanScope(common[len(common)-1])
}

func scopeFromPath(p string) string {
	p = toSlash(p)
	dir := path.Dir(p)
	if dir == "." || dir == "/" || dir == "" {
		// Use the file's base name without extension for top-level files.
		base := strings.TrimSuffix(path.Base(p), path.Ext(p))
		return cleanScope(base)
	}
	parts := strings.Split(dir, "/")
	return cleanScope(parts[len(parts)-1])
}

func inferSummary(files []git.FileChange, scope string) string {
	if len(files) == 1 {
		f := files[0]
		name := path.Base(toSlash(f.Path))
		switch f.Status {
		case "A":
			return "add " + name
		case "D":
			return "remove " + name
		case "R":
			return "rename " + path.Base(toSlash(f.OldPath)) + " to " + name
		default:
			return "update " + name
		}
	}

	target := scope
	if target == "" {
		target = pluralizeArea(files)
	}
	verb := "update"
	if allStatus(files, "A") {
		verb = "add"
	} else if allStatus(files, "D") {
		verb = "remove"
	}
	return verb + " " + target
}

// --- classification helpers ---

func isDocs(p string) bool {
	p = strings.ToLower(toSlash(p))
	if strings.HasPrefix(p, "docs/") {
		return true
	}
	ext := path.Ext(p)
	return ext == ".md" || ext == ".mdx" || ext == ".rst" || ext == ".txt"
}

func isTest(p string) bool {
	p = strings.ToLower(toSlash(p))
	base := path.Base(p)
	return strings.HasSuffix(base, "_test.go") ||
		strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") ||
		strings.HasPrefix(p, "test/") ||
		strings.HasPrefix(p, "tests/") ||
		strings.Contains(p, "/__tests__/")
}

func isCI(p string) bool {
	p = strings.ToLower(toSlash(p))
	return strings.HasPrefix(p, ".github/workflows/") ||
		strings.HasPrefix(p, ".gitlab-ci") ||
		strings.HasPrefix(p, ".circleci/") ||
		p == "azure-pipelines.yml" ||
		p == ".travis.yml"
}

func isBuild(p string) bool {
	p = strings.ToLower(toSlash(p))
	base := path.Base(p)
	switch base {
	case "dockerfile", "makefile", "go.mod", "go.sum", "package.json",
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "cargo.toml",
		"cargo.lock", "pyproject.toml", "requirements.txt", "pom.xml",
		"build.gradle", "gemfile", "gemfile.lock":
		return true
	}
	return false
}

// --- small utilities ---

func toSlash(p string) string { return strings.ReplaceAll(p, "\\", "/") }

func cleanScope(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, ".")
	return s
}

func commonPrefix(a, b []string) []string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

func allStatus(files []git.FileChange, status string) bool {
	for _, f := range files {
		if f.Status != status {
			return false
		}
	}
	return true
}

func pluralizeArea(files []git.FileChange) string {
	exts := map[string]struct{}{}
	for _, f := range files {
		ext := strings.TrimPrefix(path.Ext(toSlash(f.Path)), ".")
		if ext != "" {
			exts[ext] = struct{}{}
		}
	}
	if len(exts) == 1 {
		for e := range exts {
			return e + " files"
		}
	}
	keys := make([]string, 0, len(exts))
	for e := range exts {
		keys = append(keys, e)
	}
	sort.Strings(keys)
	return "multiple files"
}
