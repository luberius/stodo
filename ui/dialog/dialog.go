package dialog

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	text     string
	confirm  bool
	callback func(bool)
}

func New(text string, callback func(bool)) Model {
	return Model{
		text:     text,
		callback: callback,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.confirm = true
			m.callback(true)
			return m, tea.Quit
		case "n", "N", "q", "Q", "esc":
			m.confirm = false
			m.callback(false)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2)

	return style.Render(fmt.Sprintf("%s (y/n)", m.text))
}
