// Build-time website generator; it never installs software or applies configs.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
	"github.com/johnhenning/outfit/internal/sitecatalog"
	"github.com/johnhenning/outfit/internal/source"
)

func main() {
	lockPath := flag.String("lock", "release-lock.json", "Verified engine release lock")
	templatePath := flag.String("template", "site/profile-browser.html", "Website template")
	output := flag.String("output", "dist/site/tools/outfit/profiles/index.html", "Generated page")
	engine := flag.String("engine", "", "Native released CLI, checked against the lock before execution")
	flag.Parse()
	if err := run(*lockPath, *templatePath, *output, *engine); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(lockPath, templatePath, output, engine string) error {
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		return err
	}
	var lock struct {
		Version  string            `json:"version"`
		Engines  map[string]string `json:"engines"`
		Profiles struct {
			URL     string `json:"url"`
			SHA256  string `json:"sha256"`
			Version string `json:"version"`
		} `json:"profiles"`
	}
	if err = json.Unmarshal(raw, &lock); err != nil {
		return err
	}
	if lock.Version == "" || lock.Profiles.Version == "" {
		return fmt.Errorf("release versions are required")
	}
	u, err := url.Parse(lock.Profiles.URL)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return fmt.Errorf("release profile source must be an HTTPS URL")
	}
	engine, err = filepath.Abs(engine)
	if err != nil {
		return err
	}
	binary, err := os.ReadFile(engine)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(binary)
	if hex.EncodeToString(hash[:]) != lock.Engines[runtime.GOOS+"_"+runtime.GOARCH] {
		return fmt.Errorf("released CLI checksum mismatch; nothing was executed")
	}
	root, clean, err := (source.Loader{}).Open(context.Background(), lock.Profiles.URL, lock.Profiles.SHA256)
	if err != nil {
		return err
	}
	defer clean()
	c, err := catalog.Load(root)
	if err != nil {
		return err
	}
	snapshot, err := sitecatalog.Build(c, sitecatalog.Release{Engine: lock.Version, Profiles: lock.Profiles.Version, Source: lock.Profiles.URL, SHA256: lock.Profiles.SHA256}, func(name string, target platform.Target) (plan.Plan, error) {
		command := exec.Command(engine, "plan", "--source", root, "--profile", name, "--target", target.Key()+"-"+target.Arch, "--json")
		raw, err := command.Output()
		if err != nil {
			return plan.Plan{}, err
		}
		return plan.Parse(raw)
	})
	if err != nil {
		return err
	}
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	page, err := sitecatalog.Render(template, snapshot)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	return os.WriteFile(output, page, 0644)
}
