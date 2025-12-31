package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MenuItem represents a single menu item
type MenuItem struct {
	Label    string
	Category string
	Action   string
	Status   string // "ready", "pending", "error", or ""
}

// Category represents a menu category
type Category struct {
	Name     string
	Expanded bool
	Items    []MenuItem
}

// Styles
var (
	// Colors
	accentColor    = lipgloss.Color("#7C3AED") // Purple
	successColor   = lipgloss.Color("#10B981") // Green
	warningColor   = lipgloss.Color("#F59E0B") // Amber
	errorColor     = lipgloss.Color("#EF4444") // Red
	dimColor       = lipgloss.Color("#6B7280") // Gray
	highlightColor = lipgloss.Color("#A78BFA") // Light purple

	// Box styles
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Padding(0, 2).
			MarginBottom(1)

	categoryStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			PaddingLeft(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(highlightColor).
			PaddingLeft(2).
			PaddingRight(2)

	statusReadyStyle   = lipgloss.NewStyle().Foreground(successColor)
	statusPendingStyle = lipgloss.NewStyle().Foreground(warningColor)
	statusErrorStyle   = lipgloss.NewStyle().Foreground(errorColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			MarginTop(1)

	footerStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(dimColor).
			PaddingTop(0)
)

// FlatItem represents a flattened menu item for navigation
type FlatItem struct {
	Category    *Category
	Item        *MenuItem
	IsCategory  bool
	CategoryIdx int
	ItemIdx     int
}

// MenuModel represents the TUI menu state
type MenuModel struct {
	categories []Category
	flatItems  []FlatItem
	cursor     int
	choice     string
	quitting   bool
	width      int
	height     int
	showHelp   bool
}

// Key bindings
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Toggle key.Binding
	Quit   key.Binding
	Help   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("tab", " "),
		key.WithHelp("tab", "toggle"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// NewMenuModel creates a new menu model with categories
func NewMenuModel() MenuModel {
	categories := []Category{
		{
			Name:     "Setup",
			Expanded: true,
			Items: []MenuItem{
				{Label: "Doctor", Action: "Doctor (Check Environment)", Status: "ready"},
				{Label: "Init Workspace", Action: "Init Workspace", Status: "pending"},
				{Label: "Configure", Action: "Configure (Arch, Jobs...)", Status: ""},
			},
		},
		{
			Name:     "Build",
			Expanded: true,
			Items: []MenuItem{
				{Label: "Kernel Config (defconfig)", Action: "Kernel Config (defconfig)", Status: ""},
				{Label: "Kernel Menuconfig", Action: "Kernel Menuconfig (UI)", Status: ""},
				{Label: "Build Kernel", Action: "Build Kernel", Status: ""},
				{Label: "Build Modules", Action: "Build Modules", Status: ""},
				{Label: "Build Apps", Action: "Build Apps", Status: ""},
			},
		},
		{
			Name:     "Run",
			Expanded: true,
			Items: []MenuItem{
				{Label: "Run QEMU", Action: "Run QEMU", Status: ""},
				{Label: "Run QEMU (Debug)", Action: "Run QEMU (Debug Mode)", Status: ""},
			},
		},
	}

	m := MenuModel{
		categories: categories,
		width:      60,
		height:     20,
	}
	m.buildFlatItems()
	return m
}

// buildFlatItems creates a flat list for navigation
func (m *MenuModel) buildFlatItems() {
	m.flatItems = nil
	for catIdx := range m.categories {
		cat := &m.categories[catIdx]
		m.flatItems = append(m.flatItems, FlatItem{
			Category:    cat,
			IsCategory:  true,
			CategoryIdx: catIdx,
		})
		if cat.Expanded {
			for itemIdx := range cat.Items {
				m.flatItems = append(m.flatItems, FlatItem{
					Category:    cat,
					Item:        &cat.Items[itemIdx],
					IsCategory:  false,
					CategoryIdx: catIdx,
					ItemIdx:     itemIdx,
				})
			}
		}
	}
}

func (m MenuModel) Init() tea.Cmd {
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.flatItems)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.Toggle):
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.IsCategory {
					m.categories[item.CategoryIdx].Expanded = !m.categories[item.CategoryIdx].Expanded
					m.buildFlatItems()
					// Adjust cursor if needed
					if m.cursor >= len(m.flatItems) {
						m.cursor = len(m.flatItems) - 1
					}
				}
			}

		case key.Matches(msg, keys.Enter):
			if m.cursor < len(m.flatItems) {
				item := m.flatItems[m.cursor]
				if item.IsCategory {
					// Toggle category
					m.categories[item.CategoryIdx].Expanded = !m.categories[item.CategoryIdx].Expanded
					m.buildFlatItems()
				} else if item.Item != nil {
					// Select action
					m.choice = item.Item.Action
					return m, tea.Quit
				}
			}

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
		}
	}

	return m, nil
}

func (m MenuModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Title
	title := titleStyle.Render("🔧 ELMOS - Embedded Linux on MacOS")
	b.WriteString(title)
	b.WriteString("\n\n")

	// Menu items
	for i, flatItem := range m.flatItems {
		isSelected := i == m.cursor

		if flatItem.IsCategory {
			// Category header
			arrow := "▼"
			if !flatItem.Category.Expanded {
				arrow = "▶"
			}
			catText := fmt.Sprintf("%s %s", arrow, flatItem.Category.Name)
			if isSelected {
				b.WriteString(selectedStyle.Render(catText))
			} else {
				b.WriteString(categoryStyle.Render(catText))
			}
			b.WriteString("\n")
		} else if flatItem.Item != nil {
			// Menu item
			status := m.renderStatus(flatItem.Item.Status)
			label := flatItem.Item.Label

			if isSelected {
				// Selected item
				line := fmt.Sprintf("  ▶ %s", label)
				if status != "" {
					// Add padding for alignment
					padding := 40 - len(line)
					if padding < 2 {
						padding = 2
					}
					line += strings.Repeat(" ", padding) + status
				}
				b.WriteString(selectedStyle.Render(line))
			} else {
				// Normal item
				line := fmt.Sprintf("    %s", label)
				if status != "" {
					padding := 40 - len(line)
					if padding < 2 {
						padding = 2
					}
					line += strings.Repeat(" ", padding) + status
				}
				b.WriteString(itemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	// Help footer
	b.WriteString("\n")
	helpText := "↑↓: Navigate  Enter: Select  Tab: Toggle  q: Quit"
	if m.showHelp {
		helpText = `Keyboard Shortcuts:
  ↑/k     Move up
  ↓/j     Move down
  Enter   Select item / Toggle category
  Tab     Toggle category expand/collapse
  q       Quit
  ?       Toggle this help`
	}
	b.WriteString(footerStyle.Render(helpText))

	// Apply box style
	content := boxStyle.Render(b.String())
	return content
}

func (m MenuModel) renderStatus(status string) string {
	switch status {
	case "ready":
		return statusReadyStyle.Render("[✓]")
	case "pending":
		return statusPendingStyle.Render("[○]")
	case "error":
		return statusErrorStyle.Render("[✗]")
	default:
		return ""
	}
}

// Choice returns the selected action
func (m MenuModel) Choice() string {
	return m.choice
}
