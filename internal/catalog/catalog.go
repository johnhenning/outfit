// Package catalog defines the data-only contract between Outfit and profile repositories.
package catalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Install struct {
	Provider string   `yaml:"provider" json:"provider"`
	Package  string   `yaml:"package" json:"package"`
	Cask     bool     `yaml:"cask,omitempty" json:"cask,omitempty"`
	Manual   string   `yaml:"manual,omitempty" json:"manual,omitempty"`
	Arch     []string `yaml:"arch,omitempty" json:"arch,omitempty"`
}
type Package struct {
	ID       string             `yaml:"id" json:"id"`
	Name     string             `yaml:"name" json:"name"`
	Category string             `yaml:"category" json:"category"`
	Desktop  bool               `yaml:"desktop,omitempty" json:"desktop,omitempty"`
	Depends  []string           `yaml:"depends,omitempty" json:"depends,omitempty"`
	Source   string             `yaml:"source,omitempty" json:"source,omitempty"`
	Installs map[string]Install `yaml:"installs" json:"installs"`
}
type Category struct {
	ID       string   `yaml:"id" json:"id"`
	Name     string   `yaml:"name" json:"name"`
	Packages []string `yaml:"packages" json:"packages"`
}
type Group struct {
	Desktop  bool     `yaml:"desktop,omitempty" json:"desktop,omitempty"`
	Packages []string `yaml:"packages,omitempty" json:"packages,omitempty"`
	Dotfiles []string `yaml:"dotfiles,omitempty" json:"dotfiles,omitempty"`
}
type Profile struct {
	Role      string            `yaml:"role" json:"role"`
	Presets   []string          `yaml:"presets,omitempty" json:"presets,omitempty"`
	Include   []string          `yaml:"include,omitempty" json:"include,omitempty"`
	Exclude   []string          `yaml:"exclude,omitempty" json:"exclude,omitempty"`
	Dotfiles  []string          `yaml:"dotfiles,omitempty" json:"dotfiles,omitempty"`
	Preferred map[string]string `yaml:"preferred,omitempty" json:"preferred,omitempty"`
}
type Dotfile struct {
	ID        string   `yaml:"id" json:"id"`
	Source    string   `yaml:"source" json:"source"`
	Target    string   `yaml:"target" json:"target"`
	Platforms []string `yaml:"platforms,omitempty" json:"platforms,omitempty"`
	Private   bool     `yaml:"private,omitempty" json:"private,omitempty"`
	Root      string   `yaml:"-" json:"-"`
}
type Catalog struct {
	Schema     int                `yaml:"schema" json:"schema"`
	Name       string             `yaml:"name" json:"name"`
	Packages   []Package          `yaml:"packages,omitempty" json:"packages,omitempty"`
	Categories []Category         `yaml:"categories,omitempty" json:"categories,omitempty"`
	Roles      map[string]Group   `yaml:"roles,omitempty" json:"roles,omitempty"`
	Presets    map[string]Group   `yaml:"presets,omitempty" json:"presets,omitempty"`
	Profiles   map[string]Profile `yaml:"profiles,omitempty" json:"profiles,omitempty"`
	Dotfiles   []Dotfile          `yaml:"dotfiles,omitempty" json:"dotfiles,omitempty"`
	Root       string             `yaml:"-" json:"-"`
	Digest     string             `yaml:"-" json:"-"`
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.+@/-]*$`)

func SafeRelative(p string) bool {
	return p != "" && !filepath.IsAbs(p) && !strings.Contains(p, "\\") && !strings.Contains(p, ":") && p != ".." && !strings.HasPrefix(filepath.ToSlash(filepath.Clean(p)), "../")
}
func Load(root string) (Catalog, error) {
	var c Catalog
	abs, err := filepath.Abs(root)
	if err != nil {
		return c, err
	}
	raw, err := os.ReadFile(filepath.Join(abs, "outfit.yaml"))
	if err != nil {
		return c, err
	}
	d := yaml.NewDecoder(bytes.NewReader(raw))
	d.KnownFields(true)
	if err = d.Decode(&c); err != nil {
		return c, err
	}
	c.Root = abs
	for i := range c.Dotfiles {
		c.Dotfiles[i].Root = abs
	}
	if err = c.Validate(false); err != nil {
		return c, err
	}
	c.Digest, err = Digest(abs)
	return c, err
}
func Digest(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == ".git" || d.Name() == "dist" || d.Name() == ".local" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not allowed in profile bundles: %s", p)
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > 8<<20 {
			return fmt.Errorf("invalid profile file %s", p)
		}
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%d:%s:%d:", len(rel), filepath.ToSlash(rel), len(b))
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil)), err
}
func (c Catalog) Validate(complete bool) error {
	if c.Schema != 1 {
		return fmt.Errorf("unsupported profile schema %d (Outfit supports 1)", c.Schema)
	}
	if c.Name == "" {
		return fmt.Errorf("profile bundle name is required")
	}
	ids := map[string]Package{}
	for _, p := range c.Packages {
		if !identifier.MatchString(p.ID) || strings.Contains(p.ID, "/") {
			return fmt.Errorf("invalid package id %q", p.ID)
		}
		if _, ok := ids[p.ID]; ok {
			return fmt.Errorf("duplicate package %s", p.ID)
		}
		ids[p.ID] = p
		for target, i := range p.Installs {
			if target != "macos" && target != "debian" && target != "fedora" && target != "windows" {
				return fmt.Errorf("invalid platform %s", target)
			}
			switch i.Provider {
			case "brew", "apt", "dnf", "winget", "uv", "vendor", "manual":
			default:
				return fmt.Errorf("unsupported provider %q", i.Provider)
			}
			if i.Provider == "vendor" && i.Package != "uv" && i.Package != "rust" {
				return fmt.Errorf("unknown built-in vendor installer %s", i.Package)
			}
			if i.Provider == "manual" {
				if i.Manual == "" {
					return fmt.Errorf("manual package %s needs a reason", p.ID)
				}
			} else if !identifier.MatchString(i.Package) || strings.Contains(i.Package, "..") {
				return fmt.Errorf("invalid package identifier %q", i.Package)
			}
		}
	}
	visit := map[string]int{}
	var walk func(string) error
	walk = func(id string) error {
		p, ok := ids[id]
		if !ok {
			return fmt.Errorf("unknown package dependency %s", id)
		}
		if visit[id] == 1 {
			return fmt.Errorf("cyclic dependency at %s", id)
		}
		if visit[id] == 2 {
			return nil
		}
		visit[id] = 1
		for _, dep := range p.Depends {
			if err := walk(dep); err != nil {
				return err
			}
		}
		visit[id] = 2
		return nil
	}
	for id := range ids {
		if err := walk(id); err != nil {
			return err
		}
	}
	for _, cat := range c.Categories {
		for _, id := range cat.Packages {
			if _, ok := ids[id]; !ok {
				return fmt.Errorf("category %s refers to unknown package %s", cat.ID, id)
			}
		}
	}
	files := map[string]bool{}
	targets := map[string]bool{}
	for _, f := range c.Dotfiles {
		if files[f.ID] {
			return fmt.Errorf("duplicate dotfile %s", f.ID)
		}
		files[f.ID] = true
		if !SafeRelative(f.Source) || !SafeRelative(f.Target) || strings.HasPrefix(filepath.ToSlash(filepath.Clean(f.Target)), ".ssh/id_") || strings.HasSuffix(filepath.ToSlash(filepath.Clean(f.Target)), "credentials") {
			return fmt.Errorf("unsafe dotfile source/target %q -> %q", f.Source, f.Target)
		}
		canonicalTarget := filepath.ToSlash(filepath.Clean(f.Target))
		if targets[canonicalTarget] {
			return fmt.Errorf("duplicate dotfile target %s", f.Target)
		}
		targets[canonicalTarget] = true
	}
	checkGroup := func(g Group) error {
		for _, id := range g.Packages {
			if _, ok := ids[id]; !ok {
				return fmt.Errorf("unknown package %s", id)
			}
		}
		for _, id := range g.Dotfiles {
			if !files[id] {
				return fmt.Errorf("unknown dotfile %s", id)
			}
		}
		return nil
	}
	for _, g := range c.Roles {
		if err := checkGroup(g); err != nil {
			return err
		}
	}
	for _, g := range c.Presets {
		if err := checkGroup(g); err != nil {
			return err
		}
	}
	if complete {
		for name, p := range c.Profiles {
			if _, ok := c.Roles[p.Role]; !ok {
				return fmt.Errorf("profile %s has unknown role %s", name, p.Role)
			}
			for _, preset := range p.Presets {
				if _, ok := c.Presets[preset]; !ok {
					return fmt.Errorf("unknown preset %s", preset)
				}
			}
			for _, id := range append(append([]string{}, p.Include...), p.Exclude...) {
				if _, ok := ids[id]; !ok {
					return fmt.Errorf("unknown selected package %s", id)
				}
			}
			for _, id := range p.Dotfiles {
				if !files[id] {
					return fmt.Errorf("unknown dotfile %s", id)
				}
			}
			for cat, id := range p.Preferred {
				if pkg, ok := ids[id]; !ok || pkg.Category != cat {
					return fmt.Errorf("invalid preferred %s: %s", cat, id)
				}
			}
		}
	}
	return nil
}
func Merge(base, overlay Catalog) (Catalog, error) {
	out := base
	out.Packages = append(append([]Package{}, base.Packages...), overlay.Packages...)
	out.Categories = append(append([]Category{}, base.Categories...), overlay.Categories...)
	out.Dotfiles = append(append([]Dotfile{}, base.Dotfiles...), overlay.Dotfiles...)
	mergeGroups := func(a, b map[string]Group) (map[string]Group, error) {
		m := map[string]Group{}
		for k, v := range a {
			m[k] = v
		}
		for k, v := range b {
			if _, ok := m[k]; ok {
				return nil, fmt.Errorf("overlay conflicts at %s", k)
			}
			m[k] = v
		}
		return m, nil
	}
	var err error
	out.Roles, err = mergeGroups(base.Roles, overlay.Roles)
	if err != nil {
		return out, err
	}
	out.Presets, err = mergeGroups(base.Presets, overlay.Presets)
	if err != nil {
		return out, err
	}
	out.Profiles = map[string]Profile{}
	for k, v := range base.Profiles {
		out.Profiles[k] = v
	}
	for k, v := range overlay.Profiles {
		if _, ok := out.Profiles[k]; ok {
			return out, fmt.Errorf("duplicate profile %s", k)
		}
		out.Profiles[k] = v
	}
	h := sha256.Sum256([]byte(base.Digest + ":" + overlay.Digest))
	out.Digest = hex.EncodeToString(h[:])
	out.Name = base.Name + " + " + overlay.Name
	return out, out.Validate(true)
}
func (c Catalog) Package(id string) (Package, bool) {
	for _, p := range c.Packages {
		if p.ID == id {
			return p, true
		}
	}
	return Package{}, false
}
func SortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
