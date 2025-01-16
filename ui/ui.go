package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/luberius/stodo/todo"
)

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

func (t TaskItem) Title() string {
	style := lipgloss.NewStyle()
	if t.Done {
		style = style.Strikethrough(true).Foreground(lipgloss.Color("8"))
	}

	priorityStyle := style
	priorityEmoji := ""
	switch t.Priority {
	case todo.High:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("1"))
		priorityEmoji = "🔴"
	case todo.Medium:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("3"))
		priorityEmoji = "🟡"
	case todo.Low:
		priorityStyle = priorityStyle.Foreground(lipgloss.Color("2"))
		priorityEmoji = "🟢"
	}

	checkbox := map[bool]string{true: "✅", false: " "}[t.Done]
	priority := ""
	if priorityEmoji != "" {
		priority = priorityStyle.Render(priorityEmoji) + " "
	}

	return fmt.Sprintf("%s %s%s", checkbox, priority, style.Render(t.Text))
}

func (t TaskItem) Description() string { return "" }
func (t TaskItem) FilterValue() string { return t.Text }

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
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "quit"),
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

func New(store *todo.Store) Model {
	keys := newKeyMap()
	items := make([]list.Item, len(store.Tasks))
	for i, task := range store.Tasks {
		items[i] = TaskItem(task)
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)
	l := list.New(items, delegate, 0, 0)
	l.Title = "Todo"
	l.SetFilteringEnabled(false)
	l.Styles.TitleBar = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("62")).Padding(0, 1)
	l.Styles.NoItems = lipgloss.NewStyle().Margin(1, 2)
	l.SetShowHelp(false)

	ti := textinput.New()
	ti.Placeholder = "No Task"
	ti.Focus()

	return Model{
		list:      l,
		keys:      keys,
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
		switch {
		case m.dialog == addTask:
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
				cmds = append(cmds, cmd)
			}
		case m.dialog == archiveConfirm:
			switch msg.String() {
			case "y", "Y":
				m.dialog = archiveLabel
			case "n", "N", "esc":
				m.dialog = noDialog
			}
		case m.dialog == archiveLabel:
			switch msg.Type {
			case tea.KeyEnter:
				label := m.textInput.Value()
				m.store.Archive(label)
				m.textInput.Reset()
				m.dialog = noDialog
				m.quitting = true
				return m, tea.Quit
			case tea.KeyEsc:
				m.dialog = noDialog
				m.textInput.Reset()
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		default:
			switch {
			case key.Matches(msg, m.keys.Quit):
				if m.dialog == noDialog {
					m.quitting = true
					return m, tea.Quit
				} else {
					m.dialog = noDialog
					m.textInput.Reset()
				}
			case key.Matches(msg, m.keys.Add):
				m.dialog = addTask
				m.textInput.Focus()
			case key.Matches(msg, m.keys.Toggle):
				idx := m.list.Index()
				m.store.Toggle(idx)
				m.updateList()
			case key.Matches(msg, m.keys.Delete):
				idx := m.list.Index()
				m.store.Remove(idx)
				m.updateList()
			case key.Matches(msg, m.keys.Save):
				m.store.Save()
			case key.Matches(msg, m.keys.Priority):
				idx := m.list.Index()
				m.store.CyclePriority(idx)
				m.updateList()
			case key.Matches(msg, m.keys.Archive):
				m.dialog = archiveConfirm
				m.textInput.Focus()
			}
		}
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-1)
		m.dialogWidth = msg.Width / 2
		m.dialogHeight = 6
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	s := m.list.View()

	if m.dialog != noDialog {
		dialog := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Width(m.dialogWidth).
			Align(lipgloss.Center)

		var content string
		switch m.dialog {
		case addTask:
			content = fmt.Sprintf("🆕 Add New Task\n\n%s", m.textInput.View())
		case archiveConfirm:
			content = "📦 Archive todo list? (y/n)"
		case archiveLabel:
			content = fmt.Sprintf("📝 Enter archive label (optional)\n\n%s", m.textInput.View())
		}

		s = lipgloss.JoinVertical(lipgloss.Center,
			s,
			"\n",
			dialog.Render(content),
		)
	}

	return s
}

func (m *Model) updateList() {
	items := make([]list.Item, len(m.store.Tasks))
	for i, task := range m.store.Tasks {
		items[i] = TaskItem(task)
	}
	m.list.SetItems(items)
}
