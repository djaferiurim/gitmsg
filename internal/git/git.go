// Package git provides thin wrappers around the git command-line tool.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// FileChange describes a single changed path in the index.
type FileChange struct {
	Status  string // A, M, D, R, C, etc.
	Path    string
	OldPath string // populated for renames/copies
}

// Commit is a single commit summary used when generating PR descriptions.
type Commit struct {
	Hash    string
	Subject string
	Body    string
}

// Run executes git with the given arguments and returns trimmed stdout.
func Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// IsRepo reports whether the current directory is inside a git work tree.
func IsRepo() bool {
	out, err := Run("rev-parse", "--is-inside-work-tree")
	return err == nil && strings.TrimSpace(out) == "true"
}

// StageAllTracked stages modifications and deletions of already-tracked files.
func StageAllTracked() error {
	_, err := Run("add", "-u")
	return err
}

// StagedDiff returns the unified diff of staged changes.
func StagedDiff() (string, error) {
	return Run("diff", "--cached")
}

// StagedFiles returns the list of files staged for commit.
func StagedFiles() ([]FileChange, error) {
	out, err := Run("diff", "--cached", "--name-status", "-z")
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out), nil
}

// parseNameStatus parses the NUL-separated output of `git diff --name-status -z`.
func parseNameStatus(out string) []FileChange {
	parts := strings.Split(out, "\x00")
	var changes []FileChange
	for i := 0; i < len(parts); i++ {
		status := strings.TrimSpace(parts[i])
		if status == "" {
			continue
		}
		code := status[:1]
		switch code {
		case "R", "C":
			// Rename/copy entries are followed by old path then new path.
			if i+2 < len(parts) {
				changes = append(changes, FileChange{
					Status:  code,
					OldPath: parts[i+1],
					Path:    parts[i+2],
				})
				i += 2
			}
		default:
			if i+1 < len(parts) {
				changes = append(changes, FileChange{Status: code, Path: parts[i+1]})
				i++
			}
		}
	}
	return changes
}

// DiffStat returns total insertions and deletions for staged changes.
func DiffStat() (insertions, deletions int, err error) {
	out, err := Run("diff", "--cached", "--numstat")
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		add, aerr := strconv.Atoi(fields[0])
		del, derr := strconv.Atoi(fields[1])
		if aerr == nil {
			insertions += add
		}
		if derr == nil {
			deletions += del
		}
	}
	return insertions, deletions, nil
}

// CurrentBranch returns the name of the current branch.
func CurrentBranch() (string, error) {
	return Run("rev-parse", "--abbrev-ref", "HEAD")
}

// CreateCommit creates a commit with the given message.
func CreateCommit(message string) error {
	_, err := Run("commit", "-m", message)
	return err
}

// AmendCommit rewrites the most recent commit with the given message,
// keeping its existing tree plus any newly staged changes.
func AmendCommit(message string) error {
	_, err := Run("commit", "--amend", "-m", message)
	return err
}

// GitDir returns the path to the repository's .git directory.
func GitDir() (string, error) {
	out, err := Run("rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// DefaultBase guesses the base branch to compare against for a pull request.
func DefaultBase() string {
	for _, ref := range []string{"origin/main", "origin/master", "main", "master"} {
		if _, err := Run("rev-parse", "--verify", "--quiet", ref); err == nil {
			return ref
		}
	}
	return "origin/main"
}

// CommitsSince returns commits on HEAD that are not on base.
func CommitsSince(base string) ([]Commit, error) {
	rangeSpec := base + "..HEAD"
	const sep = "\x1e" // record separator
	const fieldSep = "\x1f"
	out, err := Run("log", "--pretty=format:%H"+fieldSep+"%s"+fieldSep+"%b"+sep, rangeSpec)
	if err != nil {
		return nil, err
	}
	var commits []Commit
	for _, rec := range strings.Split(out, sep) {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		fields := strings.SplitN(rec, fieldSep, 3)
		if len(fields) < 2 {
			continue
		}
		c := Commit{Hash: fields[0], Subject: fields[1]}
		if len(fields) == 3 {
			c.Body = strings.TrimSpace(fields[2])
		}
		commits = append(commits, c)
	}
	return commits, nil
}
