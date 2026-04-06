# GolfBuddy demo — run with: vhs demo.tape
# Requires: vhs (https://github.com/charmbracelet/vhs)
#           GEMINI_API_KEY set, dependencies installed

Output docs/demo.gif

Set Shell "zsh"
Set FontSize 14
Set Width 1600
Set Height 1000
Set Theme "Catppuccin Mocha"
Set PlaybackSpeed 1.0
Set TypingSpeed 50ms

# Start the TUI
Type "cd tui && go run ."
Enter
Sleep 3s

# Navigate to scan → json
Enter
Sleep 500ms
Down
Sleep 400ms
Enter
Sleep 800ms

# Load a sample JSON file — type partial path, Tab to autocomplete, then Enter
Type "tmp/emerald_golf.json"
Sleep 500ms
Enter
Sleep 3s

# Browse the scorecard table
Right
Sleep 300ms
Right
Sleep 300ms
Right
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Save and exit
Type "s"
Sleep 5s
Escape
Sleep 500ms

# Navigate to stats
Down
Sleep 400ms
Enter
Sleep 1500ms

# Browse a player
Down
Sleep 500ms
Enter
Sleep 1500ms

# Browse a round
Enter
Sleep 5s

Escape
Sleep 800ms
Escape
Sleep 800ms

# Quit
Type "q"