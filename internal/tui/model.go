package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
	"strings"
)

type Model struct {
	Catalog                                catalog.Catalog
	Target                                 platform.Target
	Config                                 plan.Config
	Stage                                  string
	Cursor, Category, Height, ReviewOffset int
	Confirmed                              bool
	Err                                    string
}

func New(c catalog.Catalog, t platform.Target, cfg plan.Config) Model {
	return Model{Catalog: c, Target: t, Config: cfg, Stage: "profiles", Height: 24}
}
func (m Model) Init() tea.Cmd { return nil }
func (m Model) items() []string {
	switch m.Stage {
	case "profiles":
		return catalog.SortedKeys(m.Catalog.Profiles)
	case "main":
		return []string{"Review changes", "Advanced software choices", "Choose role", "Choose additional presets", "Cancel"}
	case "roles":
		return catalog.SortedKeys(m.Catalog.Roles)
	case "presets":
		return catalog.SortedKeys(m.Catalog.Presets)
	case "categories":
		xs := []string{}
		for _, c := range m.Catalog.Categories {
			xs = append(xs, c.Name)
		}
		return xs
	case "packages":
		return append([]string{"None in this category"}, m.Catalog.Categories[m.Category].Packages...)
	case "review":
		return []string{"Apply this plan", "Back"}
	}
	return nil
}
func has(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
func toggle(xs []string, s string) []string {
	out := []string{}
	found := false
	for _, x := range xs {
		if x == s {
			found = true
		} else {
			out = append(out, x)
		}
	}
	if !found {
		out = append(out, s)
	}
	return out
}
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		if m.Stage == "review" && (key == "pgdown" || key == "pgup") {
			p, err := plan.Resolve(m.Catalog, m.Target, m.Config)
			if err == nil {
				step := m.Height - 14
				if step < 3 {
					step = 3
				}
				if key == "pgdown" {
					m.ReviewOffset += step
				} else {
					m.ReviewOffset -= step
				}
				maxOffset := len(p.Actions) - step
				if maxOffset < 0 {
					maxOffset = 0
				}
				if m.ReviewOffset > maxOffset {
					m.ReviewOffset = maxOffset
				}
				if m.ReviewOffset < 0 {
					m.ReviewOffset = 0
				}
			}
			return m, nil
		}
		items := m.items()
		if len(items) == 0 {
			return m, nil
		}
		if key == "up" || key == "k" {
			m.Cursor = (m.Cursor + len(items) - 1) % len(items)
		}
		if key == "down" || key == "j" {
			m.Cursor = (m.Cursor + 1) % len(items)
		}
		if key == "esc" {
			m.Stage = "main"
			m.Cursor = 0
			return m, nil
		}
		if key != "enter" && key != " " && key != "p" {
			return m, nil
		}
		selected := items[m.Cursor]
		m.Err = ""
		switch m.Stage {
		case "profiles":
			m.Config.Profile = selected
			m.Stage = "main"
			m.Cursor = 0
		case "main":
			switch m.Cursor {
			case 0:
				if _, err := plan.Resolve(m.Catalog, m.Target, m.Config); err != nil {
					m.Err = err.Error()
				} else {
					m.Stage = "review"
					m.Cursor = 1
				}
			case 1:
				m.Stage = "categories"
				m.Cursor = 0
			case 2:
				m.Stage = "roles"
				m.Cursor = 0
			case 3:
				m.Stage = "presets"
				m.Cursor = 0
			case 4:
				return m, tea.Quit
			}
		case "roles":
			m.Config.Role = selected
			m.Stage = "main"
			m.Cursor = 0
		case "presets":
			m.Config.Presets = toggle(m.Config.Presets, selected)
		case "categories":
			m.Category = m.Cursor
			m.Stage = "packages"
			m.Cursor = 0
		case "packages":
			cat := m.Catalog.Categories[m.Category]
			if m.Cursor == 0 {
				for _, id := range cat.Packages {
					if !has(m.Config.Exclude, id) {
						m.Config.Exclude = append(m.Config.Exclude, id)
					}
					if has(m.Config.Include, id) {
						m.Config.Include = toggle(m.Config.Include, id)
					}
				}
				if m.Config.Preferred != nil {
					delete(m.Config.Preferred, cat.ID)
				}
			} else {
				pkg, _ := m.Catalog.Package(selected)
				inst, ok := pkg.Installs[m.Target.Key()]
				if !ok {
					m.Err = "Unavailable on this platform"
					return m, nil
				}
				if inst.Provider == "manual" {
					m.Err = inst.Manual
					return m, nil
				}
				if key == "p" {
					if m.Config.Preferred == nil {
						m.Config.Preferred = map[string]string{}
					}
					m.Config.Preferred[cat.ID] = selected
				} else {
					if m.effective(selected) {
						if !has(m.Config.Exclude, selected) {
							m.Config.Exclude = append(m.Config.Exclude, selected)
						}
						if has(m.Config.Include, selected) {
							m.Config.Include = toggle(m.Config.Include, selected)
						}
						if m.Config.Preferred[cat.ID] == selected {
							delete(m.Config.Preferred, cat.ID)
						}
					} else {
						if !has(m.Config.Include, selected) {
							m.Config.Include = append(m.Config.Include, selected)
						}
						if has(m.Config.Exclude, selected) {
							m.Config.Exclude = toggle(m.Config.Exclude, selected)
						}
					}
				}
				if key == "p" && has(m.Config.Exclude, selected) {
					m.Config.Exclude = toggle(m.Config.Exclude, selected)
				}
			}
		case "review":
			if m.Cursor == 0 {
				m.Confirmed = true
				return m, tea.Quit
			}
			m.Stage = "main"
			m.Cursor = 0
		}
	}
	return m, nil
}
func (m Model) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("75")).Render("outfit")
	var b strings.Builder
	fmt.Fprintf(&b, "\n %s  ·  %s/%s\n\n", title, m.Target.Key(), m.Target.Arch)
	fmt.Fprintf(&b, " Profile: %s\n\n", m.Config.Profile)
	if m.Stage == "review" {
		p, err := plan.Resolve(m.Catalog, m.Target, m.Config)
		if err != nil {
			fmt.Fprintln(&b, err)
		} else {
			fmt.Fprintf(&b, " %d actions · source %s\n", len(p.Actions), m.Catalog.Name)
			limit := m.Height - 14
			if limit < 3 {
				limit = 3
			}
			end := m.ReviewOffset + limit
			if end > len(p.Actions) {
				end = len(p.Actions)
			}
			for i := m.ReviewOffset; i < end; i++ {
				a := p.Actions[i]
				flag := ""
				if a.ManualReason != "" {
					flag = " [manual]"
				}
				if a.Privileged {
					flag += " [privilege]"
				}
				fmt.Fprintf(&b, "   %s%s\n", a.Description, flag)
			}
			fmt.Fprintf(&b, " Actions %d–%d of %d · pgup/pgdn to inspect\n", min(m.ReviewOffset+1, len(p.Actions)), end, len(p.Actions))
		}
	}
	items := m.items()
	start := 0
	limit := m.Height - 11
	if limit < 4 {
		limit = 4
	}
	if m.Cursor >= limit {
		start = m.Cursor - limit + 1
	}
	for i := start; i < len(items) && i < start+limit; i++ {
		s := items[i]
		label := s
		marker := " "
		if i == m.Cursor {
			marker = ">"
		}
		if m.Stage == "packages" && i > 0 {
			pkg, _ := m.Catalog.Package(s)
			label = pkg.Name
			inst, ok := pkg.Installs[m.Target.Key()]
			if !ok {
				label += " [unavailable]"
			} else if inst.Provider == "manual" {
				label += " [manual]"
			}
			if m.effective(s) {
				label = "[x] " + label
			} else {
				label = "[ ] " + label
			}
			if m.Config.Preferred[pkg.Category] == s {
				label += " ★"
			}
		}
		if m.Stage == "presets" && has(m.Config.Presets, s) {
			label = "[x] " + label
		}
		fmt.Fprintf(&b, " %s %s\n", marker, label)
	}
	if m.Err != "" {
		fmt.Fprintf(&b, "\n %s\n", m.Err)
	}
	fmt.Fprint(&b, "\n ↑/↓ move · enter/space select · p preferred app · esc back · q cancel\n")
	if m.Stage == "presets" {
		fmt.Fprint(&b, " Select additional presets; press esc when finished.\n")
	}
	return b.String()
}
func Run(c catalog.Catalog, t platform.Target, cfg plan.Config) (plan.Config, bool, error) {
	final, err := tea.NewProgram(New(c, t, cfg), tea.WithAltScreen()).Run()
	if err != nil {
		return cfg, false, err
	}
	m := final.(Model)
	return m.Config, m.Confirmed, nil
}

func (m Model) effective(id string) bool {
	p, err := plan.Resolve(m.Catalog, m.Target, m.Config)
	if err != nil {
		return has(m.Config.Include, id) && !has(m.Config.Exclude, id)
	}
	for _, a := range p.Actions {
		if a.Kind == "package" && a.PackageID == id {
			return true
		}
	}
	return false
}
