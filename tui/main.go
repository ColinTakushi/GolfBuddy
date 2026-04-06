package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"net"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// dbg is a logger that writes to /tmp/golfbuddy.log.
// Use dbg.Printf(...) anywhere in the TUI for debug output.
// Monitor in a second terminal with: tail -f /tmp/golfbuddy.log
var dbg *log.Logger

func main() {
	f, err := os.OpenFile("/tmp/golfbuddy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not open log file: %v\n", err)
		dbg = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		defer f.Close()
		dbg = log.New(f, "", log.LstdFlags)
	}

	// Start the API server in the background.
	server := exec.Command("python3", "main.py", "api")
	server.Dir = projectRoot
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not start API server: %v\n", err)
		os.Exit(1)
	}
	defer server.Process.Kill()

	// Wait for the server port to be open
	if err := waitForPort("localhost:8000", 10*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "Server did not open port: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// waitForPort tries to connect to the given address until it succeeds or times out.
func waitForPort(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil // Port is open
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for %s", address)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
