// Package completion holds static shell completion scripts for gitmsg and
// writes the requested one to a writer.
package completion

import (
	"fmt"
	"io"
)

// Shells lists the supported shell names.
var Shells = []string{"bash", "zsh", "fish"}

const bash = `# bash completion for gitmsg
# Enable with: source <(gitmsg completion bash)
# Or install:  gitmsg completion bash > /etc/bash_completion.d/gitmsg
_gitmsg() {
    local cur prev words cword
    _init_completion 2>/dev/null || {
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
    }

    local flags="-c --commit --amend -a --all --pr --base --type --scope \
        --body --ai --no-ai --dry-run --install-hook -v --version"
    local types="feat fix docs style refactor perf test build ci chore"

    case "$prev" in
        --type)
            COMPREPLY=( $(compgen -W "$types" -- "$cur") )
            return 0
            ;;
        --base)
            COMPREPLY=( $(compgen -W "$(git branch --format='%(refname:short)' 2>/dev/null)" -- "$cur") )
            return 0
            ;;
    esac

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "completion $flags" -- "$cur") )
        return 0
    fi
    COMPREPLY=( $(compgen -W "$flags" -- "$cur") )
}
complete -F _gitmsg gitmsg
`

const zsh = `#compdef gitmsg
# zsh completion for gitmsg
# Enable with: source <(gitmsg completion zsh)
# Or install:  gitmsg completion zsh > "${fpath[1]}/_gitmsg"
_gitmsg() {
    local -a types
    types=(feat fix docs style refactor perf test build ci chore)
    _arguments -s \
        '(-c --commit)'{-c,--commit}'[create the commit using the generated message]' \
        '--amend[amend the last commit with the generated message]' \
        '(-a --all)'{-a,--all}'[stage all tracked changes before generating]' \
        '--pr[generate a pull request title and description]' \
        '--base[base branch for --pr]:branch:->branches' \
        '--type[override the commit type]:type:($types)' \
        '--scope[override the commit scope]:scope:' \
        '--body[include a body for multi-file commits]' \
        '--ai[force AI generation]' \
        '--no-ai[disable AI generation]' \
        '--dry-run[print without committing or writing files]' \
        '--install-hook[install a prepare-commit-msg hook]' \
        '(-v --version)'{-v,--version}'[print version and exit]' \
        '1:command:(completion)'

    case "$state" in
        branches)
            local -a branches
            branches=(${(f)"$(git branch --format='%(refname:short)' 2>/dev/null)"})
            _describe 'branch' branches
            ;;
    esac
}
compdef _gitmsg gitmsg
`

const fish = `# fish completion for gitmsg
# Enable with: gitmsg completion fish | source
# Or install:  gitmsg completion fish > ~/.config/fish/completions/gitmsg.fish
function __gitmsg_branches
    git branch --format='%(refname:short)' 2>/dev/null
end

complete -c gitmsg -f
complete -c gitmsg -n '__fish_use_subcommand' -a completion -d 'Output a shell completion script'
complete -c gitmsg -s c -l commit -d 'Create the commit using the generated message'
complete -c gitmsg -l amend -d 'Amend the last commit with the generated message'
complete -c gitmsg -s a -l all -d 'Stage all tracked changes before generating'
complete -c gitmsg -l pr -d 'Generate a pull request title and description'
complete -c gitmsg -l base -d 'Base branch for --pr' -x -a '(__gitmsg_branches)'
complete -c gitmsg -l type -d 'Override the commit type' -x -a 'feat fix docs style refactor perf test build ci chore'
complete -c gitmsg -l scope -d 'Override the commit scope' -x
complete -c gitmsg -l body -d 'Include a body for multi-file commits'
complete -c gitmsg -l ai -d 'Force AI generation'
complete -c gitmsg -l no-ai -d 'Disable AI generation'
complete -c gitmsg -l dry-run -d 'Print without committing or writing files'
complete -c gitmsg -l install-hook -d 'Install a prepare-commit-msg hook'
complete -c gitmsg -s v -l version -d 'Print version and exit'
`

// Write emits the completion script for the named shell to w.
func Write(w io.Writer, shell string) error {
	switch shell {
	case "bash":
		_, err := io.WriteString(w, bash)
		return err
	case "zsh":
		_, err := io.WriteString(w, zsh)
		return err
	case "fish":
		_, err := io.WriteString(w, fish)
		return err
	default:
		return fmt.Errorf("unsupported shell %q (supported: bash, zsh, fish)", shell)
	}
}
