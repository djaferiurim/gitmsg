# VHS tape for recording the gitmsg demo GIF.
#
# Install VHS (https://github.com/charmbracelet/vhs), then run:
#   vhs demo/demo.tape
# It produces demo/demo.gif, which the README references.
#
# This script builds gitmsg, creates a throwaway repo in a temp dir, and
# walks through the core commands.

Output demo/demo.gif

Require git
Require go

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 20
Set Margin 16
Set MarginFill "#1e1e2e"
Set BorderRadius 10
Set WindowBar Colorful
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set CursorBlink false

# --- Hidden setup: build a local binary and a throwaway repo. ---
Hide
Type "export PS1='$ '" Enter
Type "go build -o /tmp/gitmsg . && export PATH=/tmp:$PATH" Enter
Type "cd $(mktemp -d) && git init -q && git config user.email d@d.io && git config user.name demo" Enter
Type "mkdir -p internal/auth && printf 'package auth\n' > internal/auth/login.go && printf 'package main\n' > main.go" Enter
# Clear screen AND scrollback so the setup never appears in the recording.
Type "printf '\033[2J\033[3J\033[H'" Enter
# Wait for the clear to actually render BEFORE recording starts at Show.
# Without this, frame 0 captures the stale setup screen.
Sleep 1500ms
Show

# Let the cleared screen be the first captured frame.
Sleep 800ms

Type "# 1) Stage your work as usual" Sleep 800ms Enter
Type "git add ." Sleep 400ms Enter

Type "# 2) Let gitmsg write the message" Sleep 800ms Enter
Type "gitmsg" Sleep 500ms Enter
Sleep 2s

Type "# 3) Or stage + commit in one shot" Sleep 800ms Enter
Type "gitmsg -a -c" Sleep 500ms Enter
Sleep 2s

Type "# 4) Make a tweak — it detects a fix" Sleep 800ms Enter
Type "echo '// patch' >> internal/auth/login.go && git add ." Sleep 400ms Enter
Type "gitmsg" Sleep 500ms Enter
Sleep 3s
