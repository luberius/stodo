package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luberius/stodo/todo"
)

var styles = struct {
	title        lipgloss.Style
	item         lipgloss.Style
	selectedItem lipgloss.Style
	pagination   lipgloss.Style
	help         lipgloss.Style
	dialog       func(width int) lipgloss.Style
}{
	title:        lipgloss.NewStyle().MarginLeft(2),
	item:         lipgloss.NewStyle().PaddingLeft(4),
	selectedItem: lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")),
	pagination:   list.DefaultStyles().PaginationStyle.PaddingLeft(4),
	help:         list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1),
	dialog: func(width int) lipgloss.Style {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(width).
			Align(lipgloss.Center).
			MarginLeft(2)
	},
}

type dialogState int

const (
	noDialog dialogState = iota
	addTask
	archiveConfirm
	archiveLabel
)

type Model struct {
	list         list.Model
	keys         keyMap
	store        *todo.Store
	quitting     bool
	dialog       dialogState
	textInput    textinput.Model
	dialogWidth  int
	dialogHeight int
}

type TaskItem todo.Task

func (t TaskItem) FilterValue() string { return t.Text }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	task, ok := listItem.(TaskItem)
	if !ok {
		return
	}
	str := formatTaskItem(task, index == m.Index())
	fmt.Fprint(w, str)
}

func formatTaskItem(task TaskItem, isSelected bool) string {
	checkbox := map[bool]string{true: "✓", false: "○"}[task.Done]
	priorityEmoji := getPriorityEmoji(task.Priority)

	text := task.Text
	if task.Done {
		text = lipgloss.NewStyle().Strikethrough(true).Render(text)
	}

	str := fmt.Sprintf("%s %s %s", checkbox, priorityEmoji, text)

	if isSelected {
		return styles.selectedItem.Render("> " + strings.TrimSpace(str))
	}
	return styles.item.Render(strings.TrimSpace(str))
}

func getPriorityEmoji(priority todo.Priority) string {
	switch priority {
	case todo.High:
		return "🔴"
	case todo.Medium:
		return "🟡"
	case todo.Low:
		return "🟢"
	default:
		return ""
	}
}

func New(store *todo.Store) Model {
	items := make([]list.Item, len(store.Tasks))
	for i, task := range store.Tasks {
		items[i] = TaskItem(task)
	}

	l := list.New(items, itemDelegate{}, 0, 0)
	l.Title = "📝 Todo List"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = styles.title
	l.Styles.PaginationStyle = styles.pagination
	l.Styles.HelpStyle = styles.help

	ti := textinput.New()
	ti.Placeholder = "Enter task..."
	ti.Focus()

	return Model{
		list:      l,
		keys:      newKeyMap(),
		store:     store,
		textInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.dialog != noDialog {
			if cmd := m.handleKeyMsg(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		if cmd := m.handleKeyMsg(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-1)
		m.dialogWidth = msg.Width / 2
		m.dialogHeight = 6
	}

	if m.dialog == noDialog {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch m.dialog {
	case addTask:
		return m.handleAddTaskDialog(msg)
	case archiveLabel:
		return m.handleArchiveLabelDialog(msg)
	case archiveConfirm:
		return m.handleArchiveConfirmDialog(msg)
	}

	return m.handleNormalMode(msg)
}

func (m *Model) handleAddTaskDialog(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		if m.textInput.Value() != "" {
			m.store.Add(strings.TrimSpace(m.textInput.Value()))
			m.updateList()
			m.textInput.Reset()
			m.dialog = noDialog
		}
	case tea.KeyEsc:
		m.dialog = noDialog
		m.textInput.Reset()
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) handleArchiveLabelDialog(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEnter:
		m.store.Archive(m.textInput.Value())
		m.textInput.Reset()
		m.dialog = noDialog
	case tea.KeyEsc:
		m.dialog = noDialog
		m.textInput.Reset()
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return cmd
	}
	return nil
}

func (m *Model) handleArchiveConfirmDialog(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyRunes:
		switch string(msg.Runes) {
		case "y", "Y":
			m.dialog = archiveLabel
			m.textInput.Focus()
		case "n", "N":
			m.dialog = noDialog
		}
	case tea.KeyEsc:
		m.dialog = noDialog
	}
	return nil
}

func (m *Model) handleNormalMode(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return tea.Quit
	case key.Matches(msg, m.keys.Add):
		m.dialog = addTask
		m.textInput.Focus()
	case key.Matches(msg, m.keys.Toggle):
		m.store.Toggle(m.list.Index())
		m.updateList()
	case key.Matches(msg, m.keys.Delete):
		m.store.Remove(m.list.Index())
		m.updateList()
	case key.Matches(msg, m.keys.Save):
		m.store.Save()
	case key.Matches(msg, m.keys.Priority):
		m.store.CyclePriority(m.list.Index())
		m.updateList()
	case key.Matches(msg, m.keys.Archive):
		m.dialog = archiveConfirm
	}
	return nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.dialog != noDialog {
		dialogContent := m.getDialogContent()
		return lipgloss.NewStyle().
			MarginLeft(2).
			MarginTop(1).
			Render(styles.dialog(m.dialogWidth).Render(dialogContent))
	}

	return m.list.View()
}

func (m Model) getDialogContent() string {
	switch m.dialog {
	case addTask:
		return fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			"🆕 Add New Task",
			m.textInput.View(),
			subtleHelpStyle("enter: save • esc: cancel"),
		)
	case archiveConfirm:
		return fmt.Sprintf(
			"%s\n\n%s",
			"📦 Archive todo list?",
			subtleHelpStyle("y: yes • n: no • esc: cancel"),
		)
	case archiveLabel:
		return fmt.Sprintf(
			"%s\n\n%s\n\n%s",
			"📝 Enter archive label (optional)",
			m.textInput.View(),
			subtleHelpStyle("enter: save • esc: cancel"),
		)
	default:
		return ""
	}
}

func subtleHelpStyle(help string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(help)
}

func (m *Model) updateList() {
	items := make([]list.Item, len(m.store.Tasks))
	for i, task := range m.store.Tasks {
		items[i] = TaskItem(task)
	}
	m.list.SetItems(items)
}

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Add      key.Binding
	Toggle   key.Binding
	Quit     key.Binding
	Delete   key.Binding
	Save     key.Binding
	Priority key.Binding
	Archive  key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("→/l", "right"),
		),
		Add: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new task"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete"),
		),
		Save: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "save"),
		),
		Priority: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "priority"),
		),
		Archive: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "archive"),
		),
	}
}
