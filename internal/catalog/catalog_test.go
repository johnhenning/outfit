package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T, body string) string {
	t.Helper()
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "outfit.yaml"), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	return d
}

const minimal = `schema: 1
name: example
packages:
  - id: git
    name: Git
    category: cli
    installs:
      debian: {provider: apt, package: git}
roles:
  compute: {desktop: false, packages: [git]}
presets:
  default: {packages: [git]}
profiles:
  compute: {role: compute, presets: [default]}
`

func TestCatalogValidation(t *testing.T) {
	c, err := Load(fixture(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Packages) != 1 {
		t.Fatal("package missing")
	}
	for _, body := range []string{`schema: 9`, minimal + "unknown: true\n", minimal + "dotfiles:\n  - {source: ../private, target: .zshrc}\n", minimal + "categories: [{id: cli, name: CLI, packages: [absent]}]\n"} {
		if _, err := Load(fixture(t, body)); err == nil {
			t.Fatalf("invalid catalog accepted: %s", body)
		}
	}
}
func TestRejectDependenciesAndProviders(t *testing.T) {
	for _, body := range []string{
		`schema: 1
name: invalid
packages:
 - {id: git, name: Git, category: cli, depends: [git], installs: {debian: {provider: apt, package: git}}}
`,
		`schema: 1
name: invalid
packages:
 - {id: git, name: Git, category: cli, installs: {debian: {provider: shell, package: "touch bad"}}}
`,
	} {
		if _, err := Load(fixture(t, body)); err == nil {
			t.Fatal("unsafe catalog accepted")
		}
	}
}
func TestPersonalOverlay(t *testing.T) {
	base, err := Load(fixture(t, minimal))
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := Load(fixture(t, "schema: 1\nname: personal\nprofiles:\n  mine: {role: compute, presets: [default]}\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Merge(base, overlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Profiles["mine"]; !ok {
		t.Fatal("personal profile missing")
	}
	if _, err := Merge(base, base); err == nil {
		t.Fatal("overlay may not silently replace catalog entries")
	}
}

func TestVendorRecipesCannotBeDefinedByProfiles(t *testing.T) {
	body := `schema: 1
name: invalid
packages:
 - {id: surprise, name: Surprise, category: cli, installs: {debian: {provider: vendor, package: arbitrary-installer}}}
`
	if _, err := Load(fixture(t, body)); err == nil {
		t.Fatal("unregistered vendor installer accepted")
	}
}

func TestDotfileCannotAliasPrivateKeys(t *testing.T) {
	for _, target := range []string{".ssh/./id_ed25519", ".ssh/foo/../id_rsa", ".aws/./credentials", ".ssh/config/../id_ed25519"} {
		body := minimal + "dotfiles:\n  - {id: bad, source: public-file, target: " + target + "}\n"
		if _, err := Load(fixture(t, body)); err == nil {
			t.Fatalf("private target alias accepted: %s", target)
		}
	}
}
