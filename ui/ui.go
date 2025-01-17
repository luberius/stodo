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

const listHeight = 14

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	windowStyle       = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("#874BFD")).
				Padding(1, 2)
	dialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Align(lipgloss.Center)
)

type todoItem struct {
	task todo.Task
}

func (i todoItem) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(todoItem)
	if !ok {
		return
	}

	checkbox := map[bool]string{true: "✓", false: "○"}[i.task.Done]
	priority := getPriorityEmoji(i.task.Priority)
	str := fmt.Sprintf("%s %s %s", checkbox, priority, i.task.Text)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}
	fmt.Fprint(w, fn(str))
}

type model struct {
	list         list.Model
	store        *todo.Store
	quitting     bool
	dialog       dialogState
	textInput    textinput.Model
	dialogWidth  int
	dialogHeight int
	keys         keyMap
}

type dialogState int

const (
	noDialog dialogState = iota
	addTask
	archiveConfirm
	archiveLabel
)

func New(store *todo.Store) model {
	const defaultWidth = 50

	ti := textinput.New()
	ti.Placeholder = "Enter task..."
	ti.Focus()

	l := list.New([]list.Item{}, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Title = "Todo List"
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	m := model{
		store:     store,
		textInput: ti,
		list:      l,
		keys:      newKeyMap(),
	}
	m.updateList()
	return m
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.dialog != noDialog {
			return m.handleDialogKey(msg)
		}
		return m.handleNormalKey(msg)
	case tea.WindowSizeMsg:
		m.dialogWidth = msg.Width / 2
		m.dialogHeight = 6
		m.list.SetWidth(msg.Width - 4)
		return m, nil
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) handleDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.dialog {
	case addTask:
		switch msg.Type {
		case tea.KeyEnter:
			if m.textInput.Value() != "" {
				m.store.Add(strings.TrimSpace(m.textInput.Value()))
				m.textInput.Reset()
				m.dialog = noDialog
				m.updateList()
			}
		case tea.KeyEsc:
			m.dialog = noDialog
			m.textInput.Reset()
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	case archiveConfirm:
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
	case archiveLabel:
		switch msg.Type {
		case tea.KeyEnter:
			m.store.Archive(m.textInput.Value())
			m.textInput.Reset()
			m.dialog = noDialog
			m.updateList()
		case tea.KeyEsc:
			m.dialog = noDialog
			m.textInput.Reset()
		default:
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		m.quitting = true
		return m, tea.Quit
	case key.Matches(msg, m.keys.Add):
		m.dialog = addTask
		m.textInput.Focus()
	case key.Matches(msg, m.keys.Toggle):
		if i := m.list.Index(); i >= 0 {
			m.store.Toggle(i)
			m.updateList()
		}
	case key.Matches(msg, m.keys.Save):
		m.store.Save()
	case key.Matches(msg, m.keys.Archive):
		m.dialog = archiveConfirm
	case key.Matches(msg, m.keys.Up):
		if m.list.Index() > 0 {
			m.list.CursorUp()
		}
	case key.Matches(msg, m.keys.Down):
		if m.list.Index() < len(m.list.Items())-1 {
			m.list.CursorDown()
		}
	case key.Matches(msg, m.keys.Priority):
		if i := m.list.Index(); i >= 0 {
			m.store.CyclePriority(i)
			m.updateList()
		}
	}
	return m, nil
}

func (m *model) updateList() {
	var items []list.Item
	for _, task := range m.store.Tasks {
		items = append(items, todoItem{task: task})
	}
	m.list.SetItems(items)
}

func getPriorityEmoji(priority todo.Priority) string {
	switch priority {
	case todo.High:
		return "🚨"
	case todo.Medium:
		return "📌"
	case todo.Low:
		return "📎"
	default:
		return ""
	}
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	if m.dialog != noDialog {
		return dialogStyle.Width(m.dialogWidth).Render(m.dialogContent())
	}

	return windowStyle.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			m.list.View(),
			m.getStatusBar(),
		),
	)
}

func (m model) dialogContent() string {
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

func (m model) getStatusBar() string {
	return fmt.Sprintf(
		"n: new • space: toggle • p: priority • w: save • a: archive • q: quit",
	)
}

func subtleHelpStyle(help string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(help)
}

type keyMap struct {
	Add      key.Binding
	Toggle   key.Binding
	Quit     key.Binding
	Save     key.Binding
	Archive  key.Binding
	Up       key.Binding
	Down     key.Binding
	Priority key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
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
		Save: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "save"),
		),
		Archive: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "archive"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Priority: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "priority"),
		),
	}
}
