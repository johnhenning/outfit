// Package sitecatalog builds read-only website snapshots with the runtime resolver.
package sitecatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/johnhenning/outfit/internal/catalog"
	"github.com/johnhenning/outfit/internal/plan"
	"github.com/johnhenning/outfit/internal/platform"
)

type Release struct {
	Engine   string `json:"engine"`
	Profiles string `json:"profiles"`
	Source   string `json:"source"`
	SHA256   string `json:"sha256"`
}

type Snapshot struct {
	Release Release                         `json:"release"`
	Catalog catalog.Catalog                 `json:"catalog"`
	Plans   map[string]map[string]plan.Plan `json:"plans"`
}

func Build(c catalog.Catalog, release Release, resolve func(string, platform.Target) (plan.Plan, error)) (Snapshot, error) {
	s := Snapshot{Release: release, Catalog: c, Plans: map[string]map[string]plan.Plan{}}
	if err := c.Validate(true); err != nil {
		return s, err
	}
	if len(c.Profiles) == 0 {
		return s, fmt.Errorf("profile browser needs at least one profile")
	}
	for _, name := range catalog.SortedKeys(c.Profiles) {
		if !regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(name) {
			return s, fmt.Errorf("unsafe website profile name %q", name)
		}
		s.Plans[name] = map[string]plan.Plan{}
		for _, family := range []string{"macos", "debian", "fedora", "windows"} {
			for _, arch := range []string{"amd64", "arm64"} {
				facts := platform.Facts{Arch: arch}
				switch family {
				case "macos":
					facts.OS = "darwin"
				case "windows":
					facts.OS = "windows"
				default:
					facts.OS = "linux"
					facts.Release = "ID=" + family
				}
				target, err := platform.Detect(facts)
				if err != nil {
					return s, err
				}
				resolved, err := resolve(name, target)
				if err != nil {
					return s, fmt.Errorf("%s on %s/%s: %w", name, family, arch, err)
				}
				s.Plans[name][family+"-"+arch] = resolved
			}
		}
	}
	return s, nil
}

func Render(template []byte, snapshot Snapshot) ([]byte, error) {
	marker := []byte("__OUTFIT_PROFILE_DATA__")
	if bytes.Count(template, marker) != 1 {
		return nil, fmt.Errorf("profile browser template needs exactly one data placeholder")
	}
	// encoding/json escapes HTML characters, preventing embedded data from closing
	// its script element. The browser renders package names as text, never HTML.
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	return bytes.Replace(template, marker, raw, 1), nil
}
