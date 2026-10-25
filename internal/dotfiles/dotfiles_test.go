package dotfiles

import (
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) (catalog.Catalog, plan.Plan, string) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	os.WriteFile(filepath.Join(root, "zshrc"), []byte("new config\n"), 0600)
	return catalog.Catalog{Dotfiles: []catalog.Dotfile{{ID: "shell", Source: "zshrc", Target: ".zshrc", Root: root}}}, plan.Plan{Actions: []plan.Action{{ID: "file:shell", Kind: "dotfile", PackageID: "shell"}}}, home
}
func TestConflictsAndRepeatApply(t *testing.T) {
	c, p, home := fixture(t)
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("old config\n"), 0600)
	changes, err := Preview(c, p, home)
	if err != nil || len(changes) != 1 || !changes[0].Conflict {
		t.Fatalf("wrong preview: %+v %v", changes, err)
	}
	if _, err := Apply(changes, home, t.TempDir(), false); err == nil {
		t.Fatal("unapproved replacement accepted")
	}
	backup := t.TempDir()
	results, err := Apply(changes, home, backup, true)
	if err != nil || len(results) != 1 {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(home, ".zshrc")); string(b) != "new config\n" {
		t.Fatal("config not applied")
	}
	b, err := os.ReadFile(filepath.Join(backup, ".zshrc"))
	if err != nil || string(b) != "old config\n" {
		t.Fatal("backup missing")
	}
	changes, err = Preview(c, p, home)
	if err != nil || !changes[0].Same {
		t.Fatal("repeat apply not idempotent")
	}
}
func TestSymlinkAndStaleBoundary(t *testing.T) {
	c, p, home := fixture(t)
	outside := t.TempDir()
	os.Symlink(outside, filepath.Join(home, ".config"))
	c.Dotfiles[0].Target = ".config/test"
	if _, err := Preview(c, p, home); err == nil {
		t.Fatal("outside-home symlink allowed")
	}
	c.Dotfiles[0].Target = ".zshrc"
	changes, err := Preview(c, p, home)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("intervening edit"), 0600)
	if _, err = Apply(changes, home, t.TempDir(), true); err == nil {
		t.Fatal("stale preview applied")
	}
}
func TestPrivatePreviewDoesNotReadContent(t *testing.T) {
	c, p, home := fixture(t)
	c.Dotfiles[0].Private = true
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("a secret"), 0600)
	changes, err := Preview(c, p, home)
	if err != nil {
		t.Fatal(err)
	}
	if changes[0].Before != "" || changes[0].After != "" {
		t.Fatal("private content in diff")
	}
	if _, err = Apply(changes, home, t.TempDir(), true); err == nil {
		t.Fatal("private existing file must be merged manually, not backed up")
	}
}
