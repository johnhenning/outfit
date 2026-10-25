package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewIsReadOnly(t *testing.T) {
	root := t.TempDir()
	b := `schema: 1
name: fixture
packages:
 - {id: git, name: Git, category: cli, installs: {debian: {provider: apt, package: git}}}
roles:
 compute: {packages: [git]}
profiles:
 compute: {role: compute}
`
	os.WriteFile(filepath.Join(root, "outfit.yaml"), []byte(b), 0600)
	var out bytes.Buffer
	cmd := New("test")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"plan", "--source", root, "--profile", "compute", "--target", "debian-amd64", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"provider": "apt"`) {
		t.Fatal(out.String())
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 {
		t.Fatal("preview wrote files")
	}
}
func TestApplyRequiresConfirmation(t *testing.T) {
	cmd := New("test")
	cmd.SetArgs([]string{"apply"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("unconfirmed apply accepted")
	}
}

func TestUpdatePropagatesErrors(t *testing.T) {
	cmd := New("test")
	cmd.SetArgs([]string{"update", "--source", t.TempDir(), "--config", "missing.json"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("failed update reported success")
	}
}

func TestSetupDryRunNeedsNoTerminalAndWritesNoState(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "outfit.yaml"), []byte("schema: 1\nname: preview\nroles:\n  compute: {}\nprofiles:\n  compute: {role: compute}\n"), 0600)
	state := filepath.Join(t.TempDir(), "absent")
	var out bytes.Buffer
	cmd := New("test")
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"setup", "--source", root, "--profile", "compute", "--target", "debian-amd64", "--state-dir", state, "--dry-run", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"schema": 1`) {
		t.Fatal("dry run did not print plan")
	}
	if _, err := os.Stat(state); !os.IsNotExist(err) {
		t.Fatal("dry run wrote machine state")
	}
}
