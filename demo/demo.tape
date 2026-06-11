# VHS tape for recording the gitmsg demo GIF.
#
# All setup (building the binary, creating a throwaway repo, a clean prompt)
# is done OUTSIDE this tape by the workflow, which configures ~/.bashrc so the
# shell VHS spawns starts already inside the demo repo with gitmsg on PATH.
# This avoids VHS's `Hide` directive, which is unreliable in CI. To record
# locally, run scripts/record-demo.sh (it performs the same setup), not vhs
# directly.

Output demo/demo.gif

Require git

Set Shell "bash"
Set FontSize 18
Set Width 1100
Set Height 640
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set CursorBlink false

# The shell already starts in a clean demo repo, so the first frame is a
# bare prompt. Let it settle, then walk through the core commands.
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
