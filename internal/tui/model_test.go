package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
	"strings"
	"testing"
)

func TestCancelBeforeApply(t *testing.T) {
	m := New(catalog.Catalog{}, platform.Target{}, plan.Config{})
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if got.(Model).Confirmed {
		t.Fatal("cancel confirmed apply")
	}
}
func TestAlternativeSelection(t *testing.T) {
	c := catalog.Catalog{Profiles: map[string]catalog.Profile{"dev": {Role: "dev"}}, Roles: map[string]catalog.Group{"dev": {Desktop: true}}, Categories: []catalog.Category{{ID: "browsers", Name: "Browsers", Packages: []string{"firefox", "dia"}}}, Packages: []catalog.Package{{ID: "firefox", Name: "Firefox", Category: "browsers", Installs: map[string]catalog.Install{"macos": {Provider: "brew", Package: "firefox"}}}, {ID: "dia", Name: "Dia", Category: "browsers", Installs: map[string]catalog.Install{"macos": {Provider: "brew", Package: "dia"}}}}}
	m := New(c, platform.Target{OS: "darwin", Family: "brew", Arch: "arm64"}, plan.Config{Profile: "dev"})
	m.Stage = "packages"
	m.Category = 0
	m.Cursor = 2
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	out := got.(Model)
	if len(out.Config.Include) != 1 || out.Config.Include[0] != "dia" {
		t.Fatalf("wrong selection: %+v", out.Config)
	}
	if !strings.Contains(out.View(), "Dia") {
		t.Fatal("selected software not displayed")
	}
}

func TestReviewAllActionsReachable(t *testing.T) {
	c := catalog.Catalog{Schema: 1, Name: "long plan", Profiles: map[string]catalog.Profile{"dev": {Role: "dev"}}, Roles: map[string]catalog.Group{"dev": {}}}
	for i := 0; i < 40; i++ {
		id := fmt.Sprintf("pkg%02d", i)
		c.Packages = append(c.Packages, catalog.Package{ID: id, Name: id, Installs: map[string]catalog.Install{"debian": {Provider: "apt", Package: id}}})
		g := c.Roles["dev"]
		g.Packages = append(g.Packages, id)
		c.Roles["dev"] = g
	}
	m := New(c, platform.Target{OS: "linux", Family: "apt", Arch: "amd64"}, plan.Config{Profile: "dev"})
	m.Stage = "review"
	m.Height = 20
	for i := 0; i < 12; i++ {
		got, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
		m = got.(Model)
	}
	if !strings.Contains(m.View(), "pkg39") {
		t.Fatal("last action cannot be inspected before approval")
	}
}

func TestInheritedPackageCanBeDeselected(t *testing.T) {
	c := catalog.Catalog{Profiles: map[string]catalog.Profile{"desktop": {Role: "workstation", Include: []string{"firefox"}}}, Roles: map[string]catalog.Group{"workstation": {Desktop: true}}, Categories: []catalog.Category{{ID: "browsers", Name: "Browsers", Packages: []string{"firefox"}}}, Packages: []catalog.Package{{ID: "firefox", Name: "Firefox", Category: "browsers", Desktop: true, Installs: map[string]catalog.Install{"macos": {Provider: "brew", Package: "firefox"}}}}}
	m := New(c, platform.Target{OS: "darwin", Family: "brew", Arch: "arm64"}, plan.Config{Profile: "desktop"})
	m.Stage = "packages"
	m.Cursor = 1
	if !strings.Contains(m.View(), "[x] Firefox") {
		t.Fatal("inherited selection not shown")
	}
	got, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = got.(Model)
	p, err := plan.Resolve(c, m.Target, m.Config)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range p.Actions {
		if a.PackageID == "firefox" {
			t.Fatal("deselected inherited package still installed")
		}
	}
}
