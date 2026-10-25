package sitecatalog

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
)

func fixture() catalog.Catalog {
	installs := map[string]catalog.Install{
		"macos": {Provider: "brew", Package: "git"}, "debian": {Provider: "apt", Package: "git"},
		"fedora": {Provider: "dnf", Package: "git"}, "windows": {Provider: "winget", Package: "Git.Git"},
	}
	return catalog.Catalog{Schema: 1, Name: "Public fixture", Packages: []catalog.Package{
		{ID: "git", Name: "Git", Category: "cli", Installs: installs},
		{ID: "browser", Name: "Browser", Category: "browsers", Desktop: true, Installs: map[string]catalog.Install{
			"macos":  {Provider: "brew", Package: "browser", Cask: true},
			"debian": {Provider: "manual", Manual: "Use the official vendor installer"},
		}},
	}, Roles: map[string]catalog.Group{
		"desktop":  {Desktop: true, Packages: []string{"git", "browser"}},
		"headless": {Packages: []string{"git", "browser"}},
	}, Profiles: map[string]catalog.Profile{"workstation": {Role: "desktop"}, "dev": {Role: "headless"}}}
}

func TestSnapshotUsesResolvedPlatformAndHeadlessPlans(t *testing.T) {
	s, err := resolvedFixture(fixture(), Release{Engine: "v0.1.0-alpha.3", Profiles: "v0.1.0-alpha.2"})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Plans["workstation"]) != 8 {
		t.Fatal("all eight targets must be browsable")
	}
	mac := s.Plans["workstation"]["macos-arm64"]
	debian := s.Plans["workstation"]["debian-amd64"]
	foundManual := false
	for _, a := range debian.Actions {
		if a.PackageID == "browser" {
			foundManual = a.ManualReason != ""
		}
	}
	if !foundManual || !mac.Target.Desktop {
		t.Fatal("lost platform-specific manual/desktop behavior")
	}
	headless := s.Plans["dev"]["macos-arm64"]
	for _, a := range headless.Actions {
		if a.PackageID == "browser" {
			t.Fatal("headless browser was not skipped")
		}
	}
	if len(headless.Notes) == 0 {
		t.Fatal("resolution notes must be visible")
	}
}

func TestSnapshotRetainsReleasedResolverResults(t *testing.T) {
	s, err := Build(fixture(), Release{}, func(name string, target platform.Target) (plan.Plan, error) {
		return plan.Plan{Target: target, Notes: []string{"release-specific selection"}}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if s.Plans["dev"]["windows-arm64"].Notes[0] != "release-specific selection" {
		t.Fatal("the released resolver result was replaced")
	}
}

func TestInvalidProfileFailsGeneration(t *testing.T) {
	c := fixture()
	c.Profiles["workstation"] = catalog.Profile{Role: "unknown"}
	if _, err := resolvedFixture(c, Release{}); err == nil {
		t.Fatal("invalid data must not be published")
	}
	c.Profiles = nil
	if _, err := resolvedFixture(c, Release{}); err == nil {
		t.Fatal("an empty profile browser must not be published")
	}
}

func TestRenderEscapesScriptClosingDataAndRequiresPlaceholder(t *testing.T) {
	c := fixture()
	c.Name = "</script><img src=x onerror=alert(1)>"
	s, err := resolvedFixture(c, Release{})
	if err != nil {
		t.Fatal(err)
	}
	page, err := Render([]byte(`<script type="application/json">__OUTFIT_PROFILE_DATA__</script>`), s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(page), "</script><img") {
		t.Fatal("catalog data escaped the JSON script")
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(string(page), `<script type="application/json">`), "</script>")
	var decoded Snapshot
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil || decoded.Catalog.Name != c.Name {
		t.Fatal("data did not round trip safely", err)
	}
	if _, err := Render([]byte("missing placeholder"), s); err == nil {
		t.Fatal("must reject stale templates")
	}
}

func resolvedFixture(c catalog.Catalog, r Release) (Snapshot, error) {
	return Build(c, r, func(name string, target platform.Target) (plan.Plan, error) {
		return plan.Resolve(c, target, plan.Config{Profile: name})
	})
}
