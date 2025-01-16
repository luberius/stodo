package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luberius/stodo/todo"
	"github.com/luberius/stodo/ui"
)

func main() {
	store := todo.NewStore(".todo")
	if err := store.Load(); err != nil {
		fmt.Printf("Error loading tasks: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
