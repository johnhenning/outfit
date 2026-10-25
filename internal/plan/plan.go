// Package plan resolves declarative selections. It never executes commands.
package plan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/platform"
	"io"
	"sort"
	"strings"
)

type Config struct {
	Profile   string            `json:"profile"`
	Role      string            `json:"role,omitempty"`
	Presets   []string          `json:"presets,omitempty"`
	Include   []string          `json:"include,omitempty"`
	Exclude   []string          `json:"exclude,omitempty"`
	Dotfiles  []string          `json:"dotfiles,omitempty"`
	Preferred map[string]string `json:"preferred,omitempty"`
}
type Action struct {
	ID           string   `json:"id"`
	Kind         string   `json:"kind"`
	PackageID    string   `json:"package_id,omitempty"`
	Provider     string   `json:"provider,omitempty"`
	Description  string   `json:"description"`
	DependsOn    []string `json:"depends_on,omitempty"`
	ManualReason string   `json:"manual_reason,omitempty"`
	Privileged   bool     `json:"privileged,omitempty"`
}
type Plan struct {
	Schema       int             `json:"schema"`
	Target       platform.Target `json:"target"`
	BundleDigest string          `json:"bundle_digest"`
	ConfigDigest string          `json:"config_digest"`
	Config       Config          `json:"config"`
	Actions      []Action        `json:"actions"`
	Notes        []string        `json:"notes,omitempty"`
}

func ParseConfig(raw []byte) (Config, error) { var c Config; err := decode(raw, &c); return c, err }
func Parse(raw []byte) (Plan, error)         { var p Plan; err := decode(raw, &p); return p, err }
func decode(raw []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var tail any
	if err := d.Decode(&tail); err != io.EOF {
		return fmt.Errorf("unexpected trailing data")
	}
	return nil
}
func hash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func Resolve(c catalog.Catalog, t platform.Target, cfg Config) (Plan, error) {
	p := Plan{Schema: 1, Target: t, BundleDigest: c.Digest, Config: cfg, ConfigDigest: hash(cfg), Actions: []Action{}}
	profile, ok := c.Profiles[cfg.Profile]
	if !ok {
		return p, fmt.Errorf("unknown profile %q; available: %s", cfg.Profile, strings.Join(catalog.SortedKeys(c.Profiles), ", "))
	}
	role := profile.Role
	if cfg.Role != "" {
		role = cfg.Role
	}
	g, ok := c.Roles[role]
	if !ok {
		return p, fmt.Errorf("unknown role %s", role)
	}
	p.Target.Desktop = g.Desktop
	if t.Environment == "wsl" {
		p.Target.Desktop = false
	}
	selected := map[string]bool{}
	explicit := map[string]bool{}
	excluded := map[string]bool{}
	files := map[string]bool{}
	for _, id := range append(profile.Exclude, cfg.Exclude...) {
		if _, ok := c.Package(id); !ok {
			return p, fmt.Errorf("unknown excluded package %s", id)
		}
		excluded[id] = true
	}
	addGroup := func(g catalog.Group) {
		for _, id := range g.Packages {
			selected[id] = true
		}
		for _, id := range g.Dotfiles {
			files[id] = true
		}
	}
	addGroup(g)
	presets := append(append([]string{}, profile.Presets...), cfg.Presets...)
	for _, name := range presets {
		v, ok := c.Presets[name]
		if !ok {
			return p, fmt.Errorf("unknown preset %s", name)
		}
		addGroup(v)
	}
	for _, id := range append(profile.Include, cfg.Include...) {
		selected[id] = true
		explicit[id] = true
	}
	for _, id := range append(profile.Dotfiles, cfg.Dotfiles...) {
		files[id] = true
	}
	preferences := map[string]string{}
	for k, v := range profile.Preferred {
		preferences[k] = v
	}
	for k, v := range cfg.Preferred {
		preferences[k] = v
	}
	for cat, id := range preferences {
		pkg, ok := c.Package(id)
		if !ok || pkg.Category != cat {
			return p, fmt.Errorf("invalid preference %s=%s", cat, id)
		}
		for sid := range selected {
			other, _ := c.Package(sid)
			if other.Category == cat && !explicit[sid] {
				delete(selected, sid)
			}
		}
		selected[id] = true
		explicit[id] = true
	}
	for id := range excluded {
		delete(selected, id)
	}
	visits := map[string]int{}
	var visit func(string, bool) error
	visit = func(id string, required bool) error {
		if excluded[id] {
			return fmt.Errorf("required dependency %s was excluded", id)
		}
		if visits[id] == 2 {
			return nil
		}
		if visits[id] == 1 {
			return fmt.Errorf("cyclic package dependency %s", id)
		}
		pkg, ok := c.Package(id)
		if !ok {
			return fmt.Errorf("unknown package %s", id)
		}
		if pkg.Desktop && !p.Target.Desktop {
			if explicit[id] || required {
				return fmt.Errorf("%s requires a desktop role", id)
			}
			p.Notes = append(p.Notes, "Skipped desktop package "+id+" for headless role")
			visits[id] = 2
			return nil
		}
		inst, ok := pkg.Installs[t.Key()]
		if !ok || len(inst.Arch) > 0 && !contains(inst.Arch, t.Arch) {
			if explicit[id] || required {
				return fmt.Errorf("%s is unsupported on %s/%s", id, t.Key(), t.Arch)
			}
			p.Notes = append(p.Notes, "Unavailable on this platform: "+id)
			visits[id] = 2
			return nil
		}
		visits[id] = 1
		deps := []string{}
		for _, dep := range pkg.Depends {
			if err := visit(dep, true); err != nil {
				return err
			}
			deps = append(deps, "pkg:"+dep)
		}
		visits[id] = 2
		if inst.Provider == "brew" {
			deps = append(deps, "prereq:brew")
		}
		p.Actions = append(p.Actions, Action{ID: "pkg:" + id, Kind: "package", PackageID: id, Provider: inst.Provider, Description: "Install " + pkg.Name, DependsOn: deps, ManualReason: inst.Manual, Privileged: inst.Provider == "apt" || inst.Provider == "dnf"})
		return nil
	}
	if t.Family == "brew" {
		p.Actions = append(p.Actions, Action{ID: "prereq:brew", Kind: "prerequisite", PackageID: "brew", Provider: "vendor", Description: "Ensure Homebrew is available (install pinned official script if missing)", Privileged: true})
	}
	keys := catalog.SortedKeys(selected)
	for _, id := range keys {
		if err := visit(id, false); err != nil {
			return p, err
		}
	}
	for _, id := range catalog.SortedKeys(files) {
		var found *catalog.Dotfile
		for i := range c.Dotfiles {
			if c.Dotfiles[i].ID == id {
				found = &c.Dotfiles[i]
				break
			}
		}
		if found == nil {
			return p, fmt.Errorf("unknown dotfile %s", id)
		}
		if len(found.Platforms) > 0 && !contains(found.Platforms, t.Key()) {
			continue
		}
		p.Actions = append(p.Actions, Action{ID: "file:" + id, Kind: "dotfile", PackageID: id, Description: "Configure ~/" + found.Target})
	}
	sort.Strings(p.Notes)
	return p, nil
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
func Validate(p Plan, c catalog.Catalog, target platform.Target) error {
	if p.Schema != 1 {
		return fmt.Errorf("unsupported plan schema")
	}
	expected, err := Resolve(c, target, p.Config)
	if err != nil {
		return err
	}
	if hash(p) != hash(expected) {
		return fmt.Errorf("plan is stale or modified; regenerate it with outfit plan")
	}
	return nil
}
