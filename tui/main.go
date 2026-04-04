package main

import (
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Start the API server in the background.
	server := exec.Command("python3", "main.py", "api")
	server.Dir = projectRoot
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not start API server: %v\n", err)
	} else {
		defer server.Process.Kill()
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
