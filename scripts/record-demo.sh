#!/usr/bin/env bash
# Records demo/demo.gif without relying on VHS's flaky `Hide` directive.
#
# All setup happens here, BEFORE vhs runs:
#   - build the gitmsg binary into a temp dir on PATH
#   - create a throwaway git repo with a couple of seed files
#   - configure ~/.bashrc so the interactive shell VHS spawns starts already
#     inside that repo, with a clean "$ " prompt and gitmsg on PATH
#
# Because the shell starts clean and in the right place, the tape contains no
# setup at all and the first recorded frame is a bare prompt.
#
# Usage: scripts/record-demo.sh   (run from the repo root)
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

bindir="$(mktemp -d)"
go build -o "$bindir/gitmsg" .

demodir="$(mktemp -d)"
git -C "$demodir" init -q
git -C "$demodir" config user.email d@d.io
git -C "$demodir" config user.name demo
mkdir -p "$demodir/internal/auth"
printf 'package auth\n' > "$demodir/internal/auth/login.go"
printf 'package main\n' > "$demodir/main.go"

# Configure the interactive shell VHS spawns. Appending at the end ensures our
# settings win over the distro default ~/.bashrc (which sets its own PS1).
{
  echo "export PATH=\"$bindir:\$PATH\""
  echo "export PS1='\$ '"
  echo "cd \"$demodir\""
} >> "$HOME/.bashrc"

vhs demo/demo.tape
