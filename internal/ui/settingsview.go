package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"technews-tui/internal/config"
)

// settingsDoneMsg signals the root model that the user is done editing settings.
type settingsDoneMsg struct {
	config *config.Config
}

type settingsRowType int

const (
	rowSourceToggle settingsRowType = iota
	rowSourceSort
	rowSourceTarget
)

type settingsRow struct {
	sourceID  string
	rowType   settingsRowType
	targetIdx int // only for rowSourceTarget
}

// SettingsModel manages the dynamic source configuration view.
type SettingsModel struct {
	cfg    *config.Config
	rows   []settingsRow
	cursor int
	input  textinput.Model
	adding bool // true = text input active for a target
	width  int
	height int
}

// NewSettingsModel creates a settings view from the current config.
func NewSettingsModel(cfg *config.Config) SettingsModel {
	// Deep copy config to allow cancelling
	newCfg := &config.Config{
		Sources: make(map[string]config.SourceConfig),
		Browser: cfg.Browser,
	}
	for k, v := range cfg.Sources {
		sc := config.SourceConfig{
			Enabled: v.Enabled,
			Sort:    v.Sort,
			Targets: make([]string, len(v.Targets)),
		}
		copy(sc.Targets, v.Targets)
		newCfg.Sources[k] = sc
	}

	ti := textinput.New()
	ti.CharLimit = 200
	ti.Width = 30

	m := SettingsModel{
		cfg:   newCfg,
		input: ti,
	}
	m.rebuildRows()
	return m
}

func (m *SettingsModel) rebuildRows() {
	var rows []settingsRow

	// Fixed order for stability
	order := []string{"hn", "reddit", "lobsters", "lemmy", "devto", "rss"}

	for _, id := range order {
		sc, ok := m.cfg.Sources[id]
		if !ok {
			continue
		}

		// Row for Name + Enabled Toggle
		rows = append(rows, settingsRow{sourceID: id, rowType: rowSourceToggle})

		if sc.Enabled {
			// Row for Sort
			if len(config.ValidSorts[id]) > 0 {
				rows = append(rows, settingsRow{sourceID: id, rowType: rowSourceSort})
			}

			// Rows for Targets
			for i := range sc.Targets {
				rows = append(rows, settingsRow{sourceID: id, rowType: rowSourceTarget, targetIdx: i})
			}
		}
	}
	m.rows = rows
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 && len(m.rows) > 0 {
		m.cursor = 0
	}
}

func (m *SettingsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.adding {
			return m.updateInputMode(msg)
		}
		return m.updateBrowseMode(msg)
	}

	if m.adding {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m SettingsModel) updateBrowseMode(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	if len(m.rows) == 0 {
		if key.Matches(msg, keys.Back) {
			return m, func() tea.Msg { return settingsDoneMsg{config: m.cfg} }
		}
		return m, nil
	}

	row := m.rows[m.cursor]
	sc := m.cfg.Sources[row.sourceID]

	switch {
	case key.Matches(msg, keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, keys.Down):
		if m.cursor < len(m.rows)-1 {
			m.cursor++
		}
	case msg.String() == " ": // toggle enabled
		if row.rowType == rowSourceToggle {
			sc.Enabled = !sc.Enabled
			m.cfg.Sources[row.sourceID] = sc
			m.rebuildRows()
		}
	case msg.String() == "t": // cycle sort
		if row.rowType == rowSourceSort {
			sc.Sort = cycleNext(config.ValidSorts[row.sourceID], sc.Sort)
			m.cfg.Sources[row.sourceID] = sc
		}
	case key.Matches(msg, keys.Add): // add target
		if row.rowType == rowSourceToggle || row.rowType == rowSourceTarget {
			m.adding = true
			m.input.SetValue("")
			m.input.Focus()
			m.input.Placeholder = fmt.Sprintf("Add %s target", row.sourceID)
			return m, textinput.Blink
		}
	case key.Matches(msg, keys.Delete): // delete target
		if row.rowType == rowSourceTarget {
			sc.Targets = append(sc.Targets[:row.targetIdx], sc.Targets[row.targetIdx+1:]...)
			m.cfg.Sources[row.sourceID] = sc
			m.rebuildRows()
		}
	case key.Matches(msg, keys.Back):
		return m, func() tea.Msg {
			return settingsDoneMsg{config: m.cfg}
		}
	}
	return m, nil
}

func (m SettingsModel) updateInputMode(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			row := m.rows[m.cursor]
			sc := m.cfg.Sources[row.sourceID]
			sc.Targets = append(sc.Targets, val)
			m.cfg.Sources[row.sourceID] = sc
			m.rebuildRows()
			// Move cursor to the new item
			for i, r := range m.rows {
				if r.sourceID == row.sourceID && r.rowType == rowSourceTarget && r.targetIdx == len(sc.Targets)-1 {
					m.cursor = i
					break
				}
			}
		}
		m.adding = false
		m.input.Blur()
		return m, nil
	case "esc":
		m.adding = false
		m.input.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m SettingsModel) View() string {
	var b strings.Builder

	settingsTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6600")).
		Padding(0, 1)

	sectionLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1)

	b.WriteString(settingsTitleStyle.Render("Settings") + "\n\n")

	if len(m.rows) == 0 {
		b.WriteString("  No sources configured.\n")
	} else {
		lastSource := ""
		for i, row := range m.rows {
			if row.sourceID != lastSource {
				if lastSource != "" {
					b.WriteString("\n")
				}
				lastSource = row.sourceID
			}

			selected := i == m.cursor && !m.adding
			cur := "  "
			if selected {
				cur = selectedItemStyle.Render("> ")
			}

			sc := m.cfg.Sources[row.sourceID]

			switch row.rowType {
			case rowSourceToggle:
				status := " "
				if sc.Enabled {
					status = "x"
				}
				name := strings.ToUpper(row.sourceID)
				line := fmt.Sprintf("[%s] %s", status, name)
				if selected {
					line = lipgloss.NewStyle().Bold(true).Render(line)
				}
				b.WriteString(cur + line + "\n")
			case rowSourceSort:
				label := sectionLabelStyle.Render("  sort: ")
				val := sc.Sort
				if selected {
					val = lipgloss.NewStyle().Bold(true).Render(val) + statusBarStyle.Render(" (t to cycle)")
				}
				b.WriteString(cur + label + val + "\n")
			case rowSourceTarget:
				target := sc.Targets[row.targetIdx]
				prefix := "  - "
				if row.sourceID == "reddit" {
					prefix = "  r/"
				}
				line := prefix + target
				if selected {
					line = lipgloss.NewStyle().Bold(true).Render(line)
				}
				b.WriteString(cur + line + "\n")
			}
		}
	}

	b.WriteString("\n")

	if m.adding {
		b.WriteString("  " + m.input.View() + "\n\n")
		b.WriteString(statusBarStyle.Render("  enter confirm • esc cancel"))
	} else {
		b.WriteString(statusBarStyle.Render("  j/k navigate • space toggle • t cycle sort • a add • d delete • esc save"))
	}

	return b.String()
}

// cycleNext returns the next item in the list after current, wrapping around.
func cycleNext(list []string, current string) string {
	for i, v := range list {
		if v == current {
			return list[(i+1)%len(list)]
		}
	}
	return list[0]
}
