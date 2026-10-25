package execute

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls     []Command
	installed map[string]bool
	fail      string
}

func (f *fakeRunner) Run(ctx context.Context, c Command) (string, error) {
	f.calls = append(f.calls, c)
	if c.Args[0] == "-W" {
		if f.installed[c.Args[len(c.Args)-1]] {
			return "installed", nil
		}
		return "", &NotInstalled{}
	}
	if c.Executable == "sudo" && len(c.Args) > 0 && c.Args[0] == "apt-get" {
		id := c.Args[len(c.Args)-1]
		if id == f.fail {
			return "", errors.New("install failed")
		}
		f.installed[id] = true
		return "", nil
	}
	return "", errors.New("unexpected command")
}
func TestProviderArguments(t *testing.T) {
	pkg := catalog.Package{ID: "git", Installs: map[string]catalog.Install{"debian": {Provider: "apt", Package: "git"}}}
	cmd, err := InstallCommand(platform.Target{OS: "linux", Family: "apt"}, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Executable != "sudo" || len(cmd.Args) != 4 || cmd.Args[3] != "git" {
		t.Fatalf("wrong privileged boundary: %+v", cmd)
	}
	pkg.Installs["debian"] = catalog.Install{Provider: "apt", Package: "git; touch /tmp/bad"}
	if _, err = InstallCommand(platform.Target{OS: "linux", Family: "apt"}, pkg); err == nil {
		t.Fatal("injection accepted")
	}
}
func TestPartialFailureAndRerun(t *testing.T) {
	r := &fakeRunner{installed: map[string]bool{}, fail: "ruff"}
	c := catalog.Catalog{Packages: []catalog.Package{{ID: "git", Installs: map[string]catalog.Install{"debian": {Provider: "apt", Package: "git"}}}, {ID: "ruff", Installs: map[string]catalog.Install{"debian": {Provider: "apt", Package: "ruff"}}}}}
	p := plan.Plan{Target: platform.Target{OS: "linux", Family: "apt"}, Actions: []plan.Action{{ID: "pkg:git", Kind: "package", PackageID: "git"}, {ID: "pkg:ruff", Kind: "package", PackageID: "ruff", DependsOn: []string{"pkg:git"}}}}
	results, err := Apply(t.Context(), p, c, r, t.TempDir(), nil)
	if err == nil || len(results) != 2 || results[0].Status != "installed" || results[1].Status != "failed" {
		t.Fatalf("incorrect partial result: %+v %v", results, err)
	}
	r.fail = ""
	results, err = Apply(t.Context(), p, c, r, t.TempDir(), nil)
	if err != nil || results[0].Status != "already-present" || results[1].Status != "installed" {
		t.Fatalf("rerun not reconciled: %+v %v", results, err)
	}
}
func TestCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	r := &fakeRunner{}
	_, err := Apply(ctx, plan.Plan{Actions: []plan.Action{{ID: "pkg:git", Kind: "package"}}}, catalog.Catalog{}, r, t.TempDir(), nil)
	if !errors.Is(err, context.Canceled) || len(r.calls) != 0 {
		t.Fatal("cancelled execution ran a command")
	}
}

func TestVendorChecksum(t *testing.T) {
	data := []byte("#!/bin/sh\nexit 0\n")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()
	dir := t.TempDir()
	_, err := FetchInstaller(t.Context(), srv.Client(), Vendor{URL: srv.URL, SHA256: strings.Repeat("0", 64)}, dir)
	if err == nil {
		t.Fatal("bad installer checksum accepted")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Fatal("failed installer left executable data")
	}
	p, err := FetchInstaller(t.Context(), srv.Client(), Vendor{URL: srv.URL, SHA256: fmt.Sprintf("%x", sha256.Sum256(data))}, dir)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if !bytes.Equal(b, data) {
		t.Fatal("downloaded installer differs")
	}
}

func TestWindowsToolPaths(t *testing.T) {
	paths := ToolPaths("C:/Users/test", "C:/Users/test/AppData/Local", "windows")
	expected := filepath.Join("C:/Users/test/AppData/Local", "Microsoft", "WinGet", "Links")
	found := false
	for _, p := range paths {
		if p == expected {
			found = true
		}
	}
	if !found {
		t.Fatal("newly installed winget CLI symlink directory missing")
	}
}
