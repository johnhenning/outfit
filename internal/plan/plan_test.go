package plan

import (
	"encoding/json"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/platform"
	"strings"
	"testing"
)

func cat() catalog.Catalog {
	return catalog.Catalog{Schema: 1, Name: "test", Digest: "abc", Packages: []catalog.Package{{ID: "git", Name: "Git", Category: "cli", Installs: map[string]catalog.Install{"debian": {Provider: "apt", Package: "git"}}}, {ID: "ruff", Name: "Ruff", Category: "python", Depends: []string{"git"}, Installs: map[string]catalog.Install{"debian": {Provider: "uv", Package: "ruff"}}}, {ID: "browser", Name: "Browser", Desktop: true, Category: "browsers", Installs: map[string]catalog.Install{"debian": {Provider: "apt", Package: "firefox-esr"}}}}, Roles: map[string]catalog.Group{"compute": {Packages: []string{"git"}}}, Presets: map[string]catalog.Group{"dev": {Packages: []string{"ruff", "browser"}}}, Profiles: map[string]catalog.Profile{"compute": {Role: "compute", Presets: []string{"dev"}}}}
}
func target() platform.Target {
	return platform.Target{OS: "linux", Arch: "amd64", Family: "apt", Environment: "native"}
}
func TestResolvePlan(t *testing.T) {
	p, err := Resolve(cat(), target(), Config{Profile: "compute"})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 2 || p.Actions[0].PackageID != "git" || p.Actions[1].PackageID != "ruff" {
		t.Fatalf("wrong headless action order: %+v", p.Actions)
	}
	if !strings.Contains(strings.Join(p.Notes, " "), "browser") {
		t.Fatal("omitted desktop package must be explained")
	}
	if _, err := Resolve(cat(), target(), Config{Profile: "compute", Include: []string{"browser"}}); err == nil {
		t.Fatal("explicit incompatible desktop choice accepted")
	}
}
func TestPlanProvenance(t *testing.T) {
	p, err := Resolve(cat(), target(), Config{Profile: "compute"})
	if err != nil {
		t.Fatal(err)
	}
	p.Actions[0].Provider = "shell"
	if err := Validate(p, cat(), target()); err == nil {
		t.Fatal("tampered action accepted")
	}
	p, _ = Resolve(cat(), target(), Config{Profile: "compute"})
	c := cat()
	c.Digest = "changed"
	if Validate(p, c, target()) == nil {
		t.Fatal("stale catalog accepted")
	}
	p, _ = Resolve(cat(), target(), Config{Profile: "compute"})
	p.Config.Profile = "missing"
	if Validate(p, cat(), target()) == nil {
		t.Fatal("modified config accepted")
	}
}
func TestExcludeRequiredDependency(t *testing.T) {
	if _, err := Resolve(cat(), target(), Config{Profile: "compute", Exclude: []string{"git"}}); err == nil {
		t.Fatal("excluded required dependency accepted")
	}
}
func TestStrictConfig(t *testing.T) {
	for _, raw := range []string{`{"profile":"compute","token":"secret"}`, `{"profile":"compute"} {"profile":"other"}`} {
		if _, err := ParseConfig([]byte(raw)); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	var p Plan
	p, _ = Resolve(cat(), target(), Config{Profile: "compute"})
	b, _ := json.Marshal(p)
	if !json.Valid(b) {
		t.Fatal("plan not JSON")
	}
}

func TestSavedPlanRoundTrip(t *testing.T) {
	p, err := Resolve(cat(), target(), Config{Profile: "compute"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	saved, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(saved, cat(), target()); err != nil {
		t.Fatalf("unmodified serialized plan rejected: %v", err)
	}
}
