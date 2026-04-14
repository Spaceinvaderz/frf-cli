package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"frf-tui/internal/app"
)

func main() {
	model := app.New()
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Fatalf("app exited with error: %v", err)
	}
}