// Command gitmsg generates Conventional Commit messages and pull request
// descriptions from your staged git changes.
//
// It works fully offline using local heuristics, and optionally uses an
// OpenAI-compatible API for richer results when an API key is configured.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/djaferiurim/gitmsg/internal/ai"
	"github.com/djaferiurim/gitmsg/internal/analyze"
	"github.com/djaferiurim/gitmsg/internal/completion"
	"github.com/djaferiurim/gitmsg/internal/generate"
	"github.com/djaferiurim/gitmsg/internal/git"
)

// Version is overridden at build time via -ldflags "-X main.Version=...".
var Version = "dev"

type options struct {
	commit      bool
	amend       bool
	all         bool
	pr          bool
	base        string
	typ         string
	scope       string
	body        bool
	useAI       bool
	noAI        bool
	dryRun      bool
	installHook bool
	hookRun     bool
	version     bool
}

func main() {
	// `completion` is a subcommand, handled before flag parsing.
	if len(os.Args) > 1 && os.Args[1] == "completion" {
		if err := runCompletion(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "gitmsg: "+err.Error())
			os.Exit(1)
		}
		return
	}

	opts := parseFlags()

	if opts.version {
		fmt.Printf("gitmsg %s\n", Version)
		return
	}

	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "gitmsg: "+err.Error())
		os.Exit(1)
	}
}

// runCompletion writes a shell completion script to stdout.
func runCompletion(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gitmsg completion <bash|zsh|fish>")
	}
	return completion.Write(os.Stdout, args[0])
}

func parseFlags() options {
	var o options

	flag.BoolVar(&o.commit, "commit", false, "create the commit using the generated message")
	flag.BoolVar(&o.commit, "c", false, "create the commit using the generated message (shorthand)")
	flag.BoolVar(&o.amend, "amend", false, "amend the last commit with the generated message")
	flag.BoolVar(&o.all, "all", false, "stage all tracked changes before generating")
	flag.BoolVar(&o.all, "a", false, "stage all tracked changes before generating (shorthand)")
	flag.BoolVar(&o.pr, "pr", false, "generate a pull request title and description")
	flag.StringVar(&o.base, "base", "", "base branch for --pr (default: auto-detected)")
	flag.StringVar(&o.typ, "type", "", "override the commit type (feat, fix, docs, ...)")
	flag.StringVar(&o.scope, "scope", "", "override the commit scope")
	flag.BoolVar(&o.body, "body", true, "include a bullet-point body for multi-file commits")
	flag.BoolVar(&o.useAI, "ai", false, "force AI generation (requires an API key)")
	flag.BoolVar(&o.noAI, "no-ai", false, "disable AI generation even if a key is set")
	flag.BoolVar(&o.dryRun, "dry-run", false, "print the message without committing, amending, or writing a hook file")
	flag.BoolVar(&o.installHook, "install-hook", false, "install a prepare-commit-msg git hook")
	flag.BoolVar(&o.hookRun, "hook-run", false, "internal: invoked by the prepare-commit-msg hook")
	flag.BoolVar(&o.version, "version", false, "print version and exit")
	flag.BoolVar(&o.version, "v", false, "print version and exit (shorthand)")

	flag.Usage = usage
	flag.Parse()
	return o
}

func usage() {
	fmt.Fprint(os.Stderr, `gitmsg - generate commit messages and PR descriptions from your changes

Usage:
  gitmsg [flags]

Examples:
  gitmsg                 Print a suggested commit message for staged changes
  gitmsg -a -c           Stage tracked changes and commit with the message
  gitmsg --amend         Rewrite the last commit message from its changes
  gitmsg --type fix      Force the commit type
  gitmsg --pr            Generate a pull request title and body
  gitmsg --install-hook  Auto-fill messages on every commit
  gitmsg completion zsh  Output a shell completion script (bash|zsh|fish)

Flags:
  -c, --commit       create the commit using the generated message
      --amend        amend the last commit with the generated message
  -a, --all          stage all tracked changes before generating
      --pr           generate a pull request title and description
      --base BRANCH  base branch for --pr (default: auto-detected)
      --type TYPE    override the commit type
      --scope SCOPE  override the commit scope
      --body         include a body for multi-file commits (default true)
      --ai           force AI generation (requires GITMSG_API_KEY or OPENAI_API_KEY)
      --no-ai        disable AI generation
      --dry-run      print what would happen without committing or writing files
      --install-hook install a prepare-commit-msg hook in this repository
  -v, --version      print version and exit

Commands:
  completion SHELL   output a completion script for bash, zsh, or fish

Environment:
  GITMSG_API_KEY / OPENAI_API_KEY   enable AI generation
  GITMSG_MODEL                      model name (default: gpt-4o-mini)
  GITMSG_API_URL                    OpenAI-compatible endpoint
`)
}

func run(o options) error {
	if !git.IsRepo() {
		return fmt.Errorf("not inside a git repository")
	}

	if o.hookRun {
		return runHook(o, flag.Args())
	}
	if o.installHook {
		return runInstallHook(o)
	}

	if o.all {
		if err := git.StageAllTracked(); err != nil {
			return err
		}
	}

	useAI := decideAI(o)

	if o.pr {
		return runPR(o, useAI)
	}
	return runCommit(o, useAI)
}

func decideAI(o options) bool {
	if o.noAI {
		return false
	}
	if o.useAI {
		if !ai.Available() {
			fmt.Fprintln(os.Stderr, "gitmsg: --ai set but no API key found; falling back to offline mode")
			return false
		}
		return true
	}
	return ai.Available()
}

func runCommit(o options, useAI bool) error {
	message, err := buildMessage(o, useAI)
	if err != nil {
		return err
	}

	if o.amend {
		if o.dryRun {
			fmt.Println("Would amend with:")
			fmt.Println(indent(message))
			return nil
		}
		if err := git.AmendCommit(message); err != nil {
			return err
		}
		fmt.Println("Amended:")
		fmt.Println(indent(message))
		return nil
	}

	if o.commit {
		if o.dryRun {
			fmt.Println("Would commit with:")
			fmt.Println(indent(message))
			return nil
		}
		if err := git.CreateCommit(message); err != nil {
			return err
		}
		fmt.Println("Committed:")
		fmt.Println(indent(message))
		return nil
	}

	fmt.Println(message)
	return nil
}

// buildMessage produces a commit message from the staged changes, using AI
// when requested and falling back to local heuristics otherwise.
func buildMessage(o options, useAI bool) (string, error) {
	files, err := git.StagedFiles()
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no staged changes found; stage files with `git add` or pass -a")
	}

	insertions, deletions, _ := git.DiffStat()

	var message string
	if useAI {
		diff, derr := git.StagedDiff()
		if derr == nil {
			if msg, aerr := ai.CommitMessage(context.Background(), diff); aerr == nil {
				message = msg
			} else {
				fmt.Fprintln(os.Stderr, "gitmsg: AI generation failed ("+aerr.Error()+"); using offline mode")
			}
		}
	}

	if message == "" {
		a := analyze.Changes(files, insertions, deletions)
		if o.typ != "" {
			a.Type = o.typ
		}
		if o.scope != "" {
			a.Scope = o.scope
		}
		message = generate.CommitMessage(a, o.body)
	}
	return message, nil
}

func runPR(o options, useAI bool) error {
	base := o.base
	if base == "" {
		base = git.DefaultBase()
	}
	branch, err := git.CurrentBranch()
	if err != nil {
		return err
	}

	commits, err := git.CommitsSince(base)
	if err != nil {
		return fmt.Errorf("could not read commits against %s: %w", base, err)
	}
	if len(commits) == 0 {
		return fmt.Errorf("no commits found on %s that are not on %s", branch, base)
	}

	if useAI {
		diff, _ := git.Run("diff", base+"...HEAD")
		var subjects strings.Builder
		for _, c := range commits {
			fmt.Fprintf(&subjects, "- %s\n", c.Subject)
		}
		if out, aerr := ai.PullRequest(context.Background(), subjects.String(), diff); aerr == nil {
			fmt.Println(out)
			return nil
		} else {
			fmt.Fprintln(os.Stderr, "gitmsg: AI generation failed ("+aerr.Error()+"); using offline mode")
		}
	}

	title, body := generate.PullRequest(commits, branch)
	fmt.Println("Title: " + title)
	fmt.Println()
	fmt.Println(body)
	return nil
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "  " + l
	}
	return strings.Join(lines, "\n")
}

// runHook is invoked by the prepare-commit-msg git hook. Git passes the path
// to the commit message file and, optionally, the message source. We only
// pre-fill the message for a plain `git commit` (empty source) and never
// overwrite a message the user already supplied (-m, -t, merge, squash, etc.).
func runHook(o options, args []string) error {
	if len(args) == 0 {
		return nil // nothing to write; stay silent so commits are never blocked
	}
	msgFile := args[0]
	source := ""
	if len(args) > 1 {
		source = args[1]
	}
	if source != "" {
		return nil // message, template, merge, squash, or commit source already set
	}

	// If the user already typed something, leave it untouched.
	if existing, err := os.ReadFile(msgFile); err == nil {
		if firstNonComment(string(existing)) != "" {
			return nil
		}
	}

	message, err := buildMessage(o, decideAI(o))
	if err != nil {
		return nil // no staged changes etc.; never block the commit
	}

	if o.dryRun {
		fmt.Fprintln(os.Stderr, "gitmsg (dry-run) would pre-fill:")
		fmt.Fprintln(os.Stderr, indent(message))
		return nil
	}

	existing, _ := os.ReadFile(msgFile)
	out := message + "\n\n" + string(existing)
	return os.WriteFile(msgFile, []byte(out), 0o644)
}

// firstNonComment returns the first non-empty, non-comment line of a commit
// message buffer, used to detect whether the user already wrote something.
func firstNonComment(s string) string {
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		return t
	}
	return ""
}

const hookScript = `#!/bin/sh
# Installed by gitmsg (https://github.com/djaferiurim/gitmsg).
# Pre-fills the commit message from your staged changes.
exec gitmsg --hook-run "$1" "$2"
`

// runInstallHook writes a prepare-commit-msg hook into the repository.
func runInstallHook(o options) error {
	dir, err := git.GitDir()
	if err != nil {
		return err
	}
	hookDir := filepath.Join(dir, "hooks")
	hookPath := filepath.Join(hookDir, "prepare-commit-msg")

	if o.dryRun {
		fmt.Println("Would install prepare-commit-msg hook at " + hookPath)
		fmt.Println("Hook contents:")
		fmt.Println(indent(hookScript))
		return nil
	}

	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		return err
	}

	if data, err := os.ReadFile(hookPath); err == nil {
		if strings.Contains(string(data), "gitmsg --hook-run") {
			fmt.Println("gitmsg hook already installed at " + hookPath)
			return nil
		}
		return fmt.Errorf("a prepare-commit-msg hook already exists at %s; remove it first", hookPath)
	}

	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil {
		return err
	}
	fmt.Println("Installed prepare-commit-msg hook at " + hookPath)
	fmt.Println("gitmsg will now pre-fill messages on `git commit`.")
	return nil
}
