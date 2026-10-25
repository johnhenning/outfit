// Package dotfiles applies an explicit data-only subset of a chezmoi source tree.
// No scripts, hooks, template functions, external commands or secret evaluation run.
package dotfiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/execute"
	"github.com/johnhenning/outfit/internal/plan"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Change struct {
	ID, Target, Source, SourceHash, TargetHash string
	Before, After                              string
	Exists, Same, Conflict, Private            bool
}

func digest(b []byte) string { v := sha256.Sum256(b); return hex.EncodeToString(v[:]) }
func safePath(root, relative string) (string, error) {
	if !catalog.SafeRelative(relative) {
		return "", fmt.Errorf("unsafe relative path %s", relative)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	p := abs
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		p = filepath.Join(p, part)
		info, err := os.Lstat(p)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return "", fmt.Errorf("symlink in managed path %s", p)
			}
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return filepath.Join(abs, relative), nil
}
func readRegular(path string) ([]byte, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return nil, false, fmt.Errorf("not a small regular config file: %s", path)
	}
	b, err := os.ReadFile(path)
	return b, true, err
}
func Preview(c catalog.Catalog, p plan.Plan, home string) ([]Change, error) {
	changes := []Change{}
	for _, a := range p.Actions {
		if a.Kind != "dotfile" {
			continue
		}
		var file *catalog.Dotfile
		for i := range c.Dotfiles {
			if c.Dotfiles[i].ID == a.PackageID {
				file = &c.Dotfiles[i]
				break
			}
		}
		if file == nil {
			return nil, fmt.Errorf("unknown dotfile %s", a.PackageID)
		}
		source, err := safePath(file.Root, file.Source)
		if err != nil {
			return nil, err
		}
		b, exists, err := readRegular(source)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("missing dotfile source %s", file.Source)
		}
		target, err := safePath(home, file.Target)
		if err != nil {
			return nil, err
		}
		old, exists, err := readRegular(target)
		if err != nil {
			return nil, err
		}
		ch := Change{ID: a.ID, Target: file.Target, Source: source, SourceHash: digest(b), TargetHash: digest(old), Exists: exists, Same: exists && digest(b) == digest(old), Private: file.Private}
		ch.Conflict = exists && !ch.Same
		if !ch.Private {
			ch.Before = string(old)
			ch.After = string(b)
		}
		changes = append(changes, ch)
	}
	return changes, nil
}
func atomic(path string, b []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".outfit-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(b)
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
func Apply(changes []Change, home, backup string, replace bool) ([]execute.Result, error) {
	// Preflight the complete change set before writing any target.
	for _, ch := range changes {
		p, err := safePath(home, ch.Target)
		if err != nil {
			return nil, err
		}
		old, exists, err := readRegular(p)
		if err != nil {
			return nil, err
		}
		if exists != ch.Exists || digest(old) != ch.TargetHash {
			return nil, fmt.Errorf("target changed since preview: %s", ch.Target)
		}
		data, exists, err := readRegular(ch.Source)
		if err != nil || !exists || digest(data) != ch.SourceHash {
			return nil, fmt.Errorf("source changed since preview: %s", ch.Target)
		}
		if ch.Conflict && (!replace || ch.Private) {
			return nil, fmt.Errorf("existing %s needs %s", ch.Target, map[bool]string{true: "a manual merge of private settings", false: "--replace after reviewing the diff"}[ch.Private])
		}
	}
	results := []execute.Result{}
	for _, ch := range changes {
		if ch.Same {
			results = append(results, execute.Result{ActionID: ch.ID, Status: "already-present"})
			continue
		}
		target, err := safePath(home, ch.Target)
		if err != nil {
			return results, err
		}
		if ch.Exists {
			b, _, err := readRegular(target)
			if err != nil {
				return results, err
			}
			dst, err := safePath(backup, ch.Target)
			if err != nil {
				return results, err
			}
			if _, err = os.Stat(dst); !os.IsNotExist(err) {
				return results, fmt.Errorf("backup already exists: %s", ch.Target)
			}
			if err = atomic(dst, b, 0600); err != nil {
				return results, err
			}
		}
		b, _, err := readRegular(ch.Source)
		if err != nil {
			return results, err
		}
		if err = atomic(target, b, 0600); err != nil {
			return results, err
		}
		results = append(results, execute.Result{ActionID: ch.ID, Status: "configured"})
	}
	return results, nil
}
