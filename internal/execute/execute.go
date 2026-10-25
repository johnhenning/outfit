package execute

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Command struct {
	Executable string
	Args       []string
	Sensitive  bool
}
type Runner interface {
	Run(context.Context, Command) (string, error)
}
type RealRunner struct{ Interactive bool }
type NotInstalled struct{}

func (*NotInstalled) Error() string { return "not installed" }
func (r RealRunner) Run(ctx context.Context, c Command) (string, error) {
	cmd := exec.CommandContext(ctx, c.Executable, c.Args...)
	if r.Interactive {
		cmd.Stdin = os.Stdin
	}
	var captured bytes.Buffer
	cmd.Stdout = &captured
	cmd.Stderr = &captured
	stream := r.Interactive && (c.Executable == "sudo" || c.Executable == "bash" || c.Executable == "sh" || len(c.Args) > 0 && c.Args[0] == "install" || len(c.Args) > 1 && c.Args[0] == "tool" && c.Args[1] == "install")
	if stream {
		cmd.Stdout = io.MultiWriter(&captured, os.Stderr)
		cmd.Stderr = io.MultiWriter(&captured, os.Stderr)
	}
	err := cmd.Run()
	b := captured.Bytes()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if err != nil {
		var ex *exec.ExitError
		if errors.As(err, &ex) {
			return "", &NotInstalled{}
		}
		return "", err
	}
	if c.Sensitive {
		return "", nil
	}
	return string(b), nil
}

type Result struct {
	ActionID string `json:"action_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

var pkgIdentifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.+@/-]*$`)

func installer(t platform.Target, p catalog.Package) (catalog.Install, error) {
	i, ok := p.Installs[t.Key()]
	if !ok {
		return i, fmt.Errorf("package %s unavailable on %s", p.ID, t.Key())
	}
	if i.Provider != "manual" && (!pkgIdentifier.MatchString(i.Package) || strings.Contains(i.Package, "..")) {
		return i, fmt.Errorf("invalid package identifier")
	}
	if i.Provider != "manual" && i.Provider != "uv" && i.Provider != "vendor" && i.Provider != t.Family {
		return i, fmt.Errorf("provider %s does not match target %s", i.Provider, t.Key())
	}
	return i, nil
}
func InstallCommand(t platform.Target, p catalog.Package) (Command, error) {
	i, err := installer(t, p)
	if err != nil {
		return Command{}, err
	}
	switch i.Provider {
	case "brew":
		args := []string{"install"}
		if i.Cask {
			args = append(args, "--cask")
		}
		return Command{"brew", append(args, i.Package), false}, nil
	case "apt":
		return Command{"sudo", []string{"apt-get", "install", "-y", i.Package}, false}, nil
	case "dnf":
		return Command{"sudo", []string{"dnf", "install", "-y", i.Package}, false}, nil
	case "winget":
		return Command{"winget", []string{"install", "--id", i.Package, "--exact", "--source", "winget", "--accept-package-agreements", "--accept-source-agreements", "--disable-interactivity"}, false}, nil
	case "uv":
		return Command{"uv", []string{"tool", "install", i.Package}, false}, nil
	}
	return Command{}, fmt.Errorf("no automatic installer for %s", p.ID)
}
func installed(ctx context.Context, t platform.Target, p catalog.Package, r Runner) (bool, error) {
	i, err := installer(t, p)
	if err != nil {
		return false, err
	}
	var c Command
	switch i.Provider {
	case "manual":
		return false, nil
	case "vendor":
		name := i.Package
		if name == "rust" {
			name = "rustup"
		}
		if name != "uv" && name != "rustup" {
			return false, fmt.Errorf("unknown vendor %s", name)
		}
		_, err := r.Run(ctx, Command{name, []string{"--version"}, false})
		return err == nil, nil
	case "brew":
		args := []string{"list", "--versions"}
		if i.Cask {
			args = append(args, "--cask")
		}
		c = Command{"brew", append(args, i.Package), false}
	case "apt":
		c = Command{"dpkg-query", []string{"-W", "-f=${db:Status-Status}", i.Package}, false}
	case "dnf":
		c = Command{"rpm", []string{"-q", i.Package}, false}
	case "winget":
		c = Command{"winget", []string{"list", "--id", i.Package, "--exact", "--source", "winget", "--accept-source-agreements", "--disable-interactivity"}, false}
	case "uv":
		c = Command{"uv", []string{"tool", "list"}, false}
	default:
		return false, fmt.Errorf("unknown provider %s", i.Provider)
	}
	out, err := r.Run(ctx, c)
	if err != nil {
		var missing *NotInstalled
		if errors.As(err, &missing) {
			return false, nil
		}
		return false, err
	}
	if i.Provider == "apt" {
		return strings.TrimSpace(out) == "installed", nil
	}
	if i.Provider == "uv" {
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(line, i.Package+" ") {
				return true, nil
			}
		}
		return false, nil
	}
	return strings.TrimSpace(out) != "", nil
}

type StateStore string

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".outfit-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Apply requires caller validation of plan provenance. It rechecks installed state per action.
func Apply(ctx context.Context, p plan.Plan, c catalog.Catalog, r Runner, state string, notify func(Result)) ([]Result, error) {
	results := []Result{}
	successful := map[string]bool{}
	var runErr error
	journal := filepath.Join(state, fmt.Sprintf("run-%d.json", time.Now().UnixNano()))
	for _, a := range p.Actions {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		res := Result{ActionID: a.ID}
		blocked := false
		for _, d := range a.DependsOn {
			if !successful[d] {
				blocked = true
			}
		}
		if blocked {
			res.Status = "skipped"
			res.Message = "prerequisite did not complete"
		} else if a.Kind == "prerequisite" && a.PackageID == "brew" {
			_, err := r.Run(ctx, Command{"brew", []string{"--version"}, false})
			if err == nil {
				res.Status = "already-present"
			} else if err = installVendor(ctx, "brew", r); err != nil {
				res.Status = "failed"
				res.Message = "Homebrew prerequisite did not complete"
				runErr = err
			} else {
				res.Status = "installed"
			}
		} else if a.Kind == "dotfile" {
			res.Status = "manual"
			res.Message = "Apply through outfit dotfiles after reviewing its diff"
		} else if a.Kind != "package" {
			return results, fmt.Errorf("unknown action kind %s", a.Kind)
		} else {
			pkg, ok := c.Package(a.PackageID)
			if !ok {
				return results, fmt.Errorf("unknown package %s", a.PackageID)
			}
			i, err := installer(p.Target, pkg)
			if err != nil {
				return results, err
			}
			if i.Provider == "manual" {
				res.Status = "manual"
				res.Message = i.Manual
			} else {
				present, err := installed(ctx, p.Target, pkg, r)
				if err == nil && present {
					res.Status = "already-present"
				} else if err == nil {
					var cmd Command
					if i.Provider == "vendor" {
						err = installVendor(ctx, i.Package, r)
					} else {
						cmd, err = InstallCommand(p.Target, pkg)
					}
					if err == nil && i.Provider != "vendor" {
						_, err = r.Run(ctx, cmd)
					}
					if err == nil {
						res.Status = "installed"
						RefreshPath()
					}
				}
				if err != nil {
					res.Status = "failed"
					res.Message = fmt.Sprintf("%s failed; rerun outfit doctor and check the package manager", pkg.ID)
					runErr = fmt.Errorf("one or more install actions failed")
				}
			}
		}
		successful[a.ID] = res.Status == "installed" || res.Status == "already-present"
		results = append(results, res)
		if err := writeJSON(journal, struct {
			Plan    plan.Plan `json:"plan"`
			Results []Result  `json:"results"`
		}{p, results}); err != nil {
			return results, fmt.Errorf("save execution journal: %w", err)
		}
		if notify != nil {
			notify(res)
		}
	}
	return results, runErr
}
func Save(path string, v any) error { return writeJSON(path, v) }
